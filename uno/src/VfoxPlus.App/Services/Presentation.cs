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

using VfoxPlus.Core.Vfox;

namespace VfoxPlus.App.Services;

/// <summary>What the user interface says about things, in the language of the user.</summary>
internal static class Presentation
{
    /// <summary>Names a task: what vfox is doing.</summary>
    /// <param name="command">The command.</param>
    /// <param name="t">Returns a message.</param>
    /// <returns>The title.</returns>
    public static string TaskTitle(VfoxCommand command, Func<string, (string Name, object? Value)[], string> t)
    {
        string subject = command.Subject;
        return command.Operation switch
        {
            VfoxOperation.Install => t("taskInstall", [("subject", subject)]),
            VfoxOperation.Uninstall => t("taskUninstall", [("subject", subject)]),
            VfoxOperation.Use => t("taskUse", [("subject", subject)]),
            VfoxOperation.Unuse => t("taskUnuse", [("subject", subject)]),
            VfoxOperation.AddPlugin => subject.Length == 0 ? t("taskAddPluginFromSource", []) : t("taskAddPlugin", [("subject", subject)]),
            VfoxOperation.RemovePlugin => t("taskRemovePlugin", [("subject", subject)]),
            VfoxOperation.UpdatePlugin => t("taskUpdatePlugin", [("subject", subject)]),
            _ => t("taskUpdateAllPlugins", []),
        };
    }

    /// <summary>Says how far a task that is queued or running has come.</summary>
    /// <param name="task">The task.</param>
    /// <param name="t">Returns a message.</param>
    /// <returns>The status line.</returns>
    public static string Progress(State.VfoxTask task, Func<string, (string Name, object? Value)[], string> t)
    {
        if (task.Progress is { } progress)
        {
            return t("taskDownloading", [("percent", (int)Math.Round(progress * 100))]);
        }

        return t(task.Status == State.VfoxTaskStatus.Queued ? "taskStatusQueued" : "taskStatusRunning", []);
    }

    /// <summary>Returns the message of an error that the user can read.</summary>
    /// <param name="error">The error.</param>
    /// <returns>The message.</returns>
    public static string Describe(Exception error) => error.Message;
}
