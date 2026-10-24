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

#include "my_application.h"

int main(int argc, char** argv) {
  // Draw on the processor instead of OpenGL: without a real GPU, OpenGL means
  // Mesa's software rasterizer and its compiler, which take as much memory as
  // the rest of the application (see docs/memory.md). The engine draws with
  // Skia then (my_application.cc), and GDK is told not to open OpenGL either.
  g_setenv("FLUTTER_LINUX_RENDERER", "software", FALSE);
  g_setenv("GDK_GL", "disable", FALSE);
  g_autoptr(MyApplication) app = my_application_new();
  return g_application_run(G_APPLICATION(app), argc, argv);
}
