// Copyright 2026 Han Li and contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import 'dart:async';
import 'dart:collection';

import 'package:flutter/foundation.dart';

import '../core/process_runner.dart';
import '../core/terminal_text.dart';
import '../core/vfox/vfox_cli.dart';
import '../core/vfox/vfox_command.dart';
import '../core/vfox/vfox_output.dart';

/// The life cycle of a [VfoxTask].
enum TaskStatus {
  /// Waiting for earlier tasks to finish.
  queued,

  /// The vfox process is running.
  running,

  /// vfox exited with code zero.
  succeeded,

  /// vfox could not be started or exited with a non-zero code.
  failed,

  /// The user cancelled the task.
  cancelled;

  /// Whether the task has not finished yet.
  bool get isActive => this == queued || this == running;
}

/// One execution of a state-changing [VfoxCommand].
class VfoxTask extends ChangeNotifier {
  VfoxTask._(this.id, this.command, this.createdAt, this._maxLogLines);

  /// Sequential identifier, unique within a [TaskQueue].
  final int id;

  /// The command that is executed.
  final VfoxCommand command;

  /// When the task was queued.
  final DateTime createdAt;

  final int _maxLogLines;
  final ListQueue<String> _log = ListQueue<String>();
  final List<String> _stdout = <String>[];
  final List<String> _stderr = <String>[];
  int _droppedLines = 0;
  RunningProcess? _process;

  TaskStatus _status = TaskStatus.queued;

  /// The current status.
  TaskStatus get status => _status;

  double? _progress;

  /// Download progress between 0 and 1, if vfox reported any.
  double? get progress => _progress;

  DateTime? _startedAt;

  /// When the process started.
  DateTime? get startedAt => _startedAt;

  DateTime? _finishedAt;

  /// When the task finished.
  DateTime? get finishedAt => _finishedAt;

  int? _exitCode;

  /// The exit code of vfox, if it ran to completion.
  int? get exitCode => _exitCode;

  String? _errorMessage;

  /// A short description of the failure, if the task failed.
  String? get errorMessage => _errorMessage;

  /// Output lines of the process (standard output and error interleaved),
  /// without progress bar redraws. Only the most recent lines are kept.
  List<String> get log => List<String>.unmodifiable(_log);

  /// How many old lines were dropped from [log].
  int get droppedLines => _droppedLines;

  /// The text that summarises a successful run, usually its last line.
  String? get lastLine => _log.isEmpty ? null : _log.last;

  void _append(String line, {required bool isError}) {
    final double? progress = parseDownloadProgress(line);
    if (progress != null) {
      _progress = progress;
    } else {
      (isError ? _stderr : _stdout).add(line);
      _log.addLast(line);
      if (_log.length > _maxLogLines) {
        _log.removeFirst();
        _droppedLines++;
      }
    }
    notifyListeners();
  }

  void _setStatus(TaskStatus status) {
    _status = status;
    notifyListeners();
  }
}

/// Runs state-changing vfox commands one after another.
///
/// vfox does not lock its data directory, so concurrent installs, uses and
/// removals could corrupt `.vfox.toml` or the symlinks. The queue therefore
/// runs at most one command at a time, in the order they were enqueued.
class TaskQueue extends ChangeNotifier {
  /// Creates a queue that executes commands with `cli`.
  TaskQueue({required this._cli, this.maxLogLines = 2000, this.maxFinishedTasks = 50});

  final VfoxCli _cli;

  /// How many log lines each task keeps.
  final int maxLogLines;

  /// How many finished tasks are kept before the oldest are forgotten.
  final int maxFinishedTasks;

  final List<VfoxTask> _tasks = <VfoxTask>[];
  final StreamController<VfoxTask> _finished = StreamController<VfoxTask>.broadcast();
  int _nextId = 1;
  VfoxTask? _running;
  bool _disposed = false;

  /// All known tasks, newest first.
  List<VfoxTask> get tasks => List<VfoxTask>.unmodifiable(_tasks.reversed);

  /// The number of queued and running tasks.
  int get activeCount => _tasks.where((VfoxTask task) => task.status.isActive).length;

  /// Whether any finished task can be cleared.
  bool get hasFinished => _tasks.any((VfoxTask task) => !task.status.isActive);

  /// Emits every task once it has finished, failed or been cancelled.
  Stream<VfoxTask> get onFinished => _finished.stream;

  /// Queues [command] and returns its task.
  ///
  /// If an identical command is already queued or running, its task is
  /// returned instead of queueing the command twice.
  VfoxTask enqueue(VfoxCommand command) {
    for (final VfoxTask task in _tasks) {
      if (task.status.isActive && _sameCommand(task.command, command)) {
        return task;
      }
    }
    final task = VfoxTask._(_nextId++, command, DateTime.now(), maxLogLines);
    _tasks.add(task);
    _trimFinished();
    notifyListeners();
    _pump();
    return task;
  }

  /// Queues the command of [task] again.
  VfoxTask retry(VfoxTask task) => enqueue(task.command);

  /// Cancels [task]: a queued task is dropped, a running process is killed.
  void cancel(VfoxTask task) {
    switch (task.status) {
      case TaskStatus.queued:
        _finish(task, TaskStatus.cancelled);
      case TaskStatus.running:
        task._process?.kill();
        // The status is settled in _execute once the process has exited.
        task._errorMessage = null;
        task._setStatus(TaskStatus.cancelled);
      case TaskStatus.succeeded || TaskStatus.failed || TaskStatus.cancelled:
        break;
    }
  }

  /// Forgets every finished task.
  void clearFinished() {
    _tasks.removeWhere((VfoxTask task) => !task.status.isActive);
    notifyListeners();
  }

  /// The active task of [operation] for [sdk] (and [version]), if any.
  VfoxTask? activeTask({VfoxOperation? operation, String? sdk, String? version}) {
    for (final VfoxTask task in _tasks) {
      if (task.status.isActive &&
          (operation == null || task.command.operation == operation) &&
          (sdk == null || task.command.sdk == sdk) &&
          (version == null || task.command.version == version)) {
        return task;
      }
    }
    return null;
  }

  void _pump() {
    if (_running != null || _disposed) {
      return;
    }
    final VfoxTask? next = _tasks.where((VfoxTask task) => task.status == TaskStatus.queued).firstOrNull;
    if (next == null) {
      return;
    }
    _running = next;
    unawaited(_execute(next));
  }

  Future<void> _execute(VfoxTask task) async {
    task
      .._startedAt = DateTime.now()
      .._setStatus(TaskStatus.running);
    notifyListeners();
    try {
      final RunningProcess process = await _cli.start(task.command);
      task._process = process;
      if (task.status == TaskStatus.cancelled) {
        // Cancelled while the process was being spawned.
        process.kill();
      }
      await Future.wait(<Future<void>>[
        terminalLines(process.stdout).forEach((String line) => task._append(line, isError: false)),
        terminalLines(process.stderr).forEach((String line) => task._append(line, isError: true)),
      ]);
      final int exitCode = await process.exitCode;
      task._exitCode = exitCode;
      if (task.status == TaskStatus.cancelled) {
        _finish(task, TaskStatus.cancelled);
      } else if (exitCode == 0) {
        _finish(task, TaskStatus.succeeded);
      } else {
        final String summary = summarizeError(task._stdout.join('\n'), task._stderr.join('\n'));
        task._errorMessage = summary.isEmpty ? 'vfox exited with code $exitCode' : summary;
        _finish(task, TaskStatus.failed);
      }
    } on VfoxException catch (error) {
      task._errorMessage = error.message;
      _finish(task, TaskStatus.failed);
    } finally {
      task._process = null;
      _running = null;
      _pump();
    }
  }

  void _finish(VfoxTask task, TaskStatus status) {
    task
      .._finishedAt = DateTime.now()
      .._setStatus(status);
    if (!_disposed) {
      notifyListeners();
      _finished.add(task);
    }
  }

  void _trimFinished() {
    final List<VfoxTask> finished = _tasks.where((VfoxTask task) => !task.status.isActive).toList();
    final int excess = finished.length - maxFinishedTasks;
    if (excess > 0) {
      _tasks.removeWhere(finished.take(excess).toSet().contains);
    }
  }

  static bool _sameCommand(VfoxCommand a, VfoxCommand b) =>
      a.workingDirectory == b.workingDirectory && listEquals(a.arguments, b.arguments);

  @override
  void dispose() {
    _disposed = true;
    for (final VfoxTask task in _tasks) {
      task._process?.kill();
    }
    unawaited(_finished.close());
    super.dispose();
  }
}
