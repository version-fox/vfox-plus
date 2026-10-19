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

// Prints the name of every assembly that a .mstat file references, one per
// line, sorted: the assemblies that an ahead-of-time (Native AOT) publish of
// a .NET application kept after trimming, which `-p:IlcGenerateMstatFile=true`
// writes as `<name>.mstat` next to the published binary. A .mstat file is
// itself a .NET assembly, with an AssemblyRef table entry for every assembly
// whose types, fields or methods survived trimming (see the Uno Platform
// documentation of Native AOT); this reads that table with
// System.Reflection.Metadata, part of the shared framework, so this project
// has no package dependency and needs no network to restore or build.
//
// Usage: dotnet run -- <path to .mstat>, or build once and run the binary.

using System.Reflection.Metadata;
using System.Reflection.PortableExecutable;

if (args.Length != 1)
{
    Console.Error.WriteLine("Usage: MstatAssemblies <path to .mstat>");
    return 2;
}

using var stream = File.OpenRead(args[0]);
using var peReader = new PEReader(stream);
var reader = peReader.GetMetadataReader();

var names = new SortedSet<string>(StringComparer.Ordinal);
foreach (var handle in reader.AssemblyReferences)
{
    names.Add(reader.GetString(reader.GetAssemblyReference(handle).Name));
}
foreach (var name in names)
{
    Console.WriteLine(name);
}
return 0;
