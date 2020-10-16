using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Reflection;
using System.Runtime.Loader;
using Fission.DotNetCore.Model;
using Fission.DotNetCore.Utilty;
using Microsoft.CodeAnalysis;
using Microsoft.CodeAnalysis.CSharp;
using Microsoft.CodeAnalysis.Emit;

namespace Fission.DotNetCore.Compiler
{
    //adapted from this article http://www.tugberkugurlu.com/archive/compiling-c-sharp-code-into-memory-and-executing-it-with-roslyn
    class FissionCompiler
    {
        string packagepath = string.Empty;
        FunctionSpecification functionSpecification = null;
        public FissionCompiler(string _packagePath)
        {
            this.packagepath = _packagePath;
        }
        public static Function Compile(string code, out List<string> errors)
        {
            errors = new List<string>();

            SyntaxTree syntaxTree = CSharpSyntaxTree.ParseText(code);

            string assemblyName = Path.GetRandomFileName();

            var coreDir = Directory.GetParent(typeof(Enumerable).GetTypeInfo().Assembly.Location);      
            
            List<MetadataReference> references = new List<MetadataReference>
            {
                MetadataReference.CreateFromFile(coreDir.FullName + Path.DirectorySeparatorChar + "mscorlib.dll"),
                MetadataReference.CreateFromFile(typeof(object).GetTypeInfo().Assembly.Location),
                MetadataReference.CreateFromFile(Assembly.GetEntryAssembly().Location),
                MetadataReference.CreateFromFile(typeof(System.Runtime.Serialization.Json.DataContractJsonSerializer).GetTypeInfo().Assembly.Location)
            };

            foreach (var referencedAssembly in Assembly.GetEntryAssembly().GetReferencedAssemblies())
            {
                var assembly = Assembly.Load(referencedAssembly);
                references.Add(MetadataReference.CreateFromFile(assembly.Location));
            }

            CSharpCompilation compilation = CSharpCompilation.Create(
                assemblyName,
                syntaxTrees: new[] { syntaxTree },
                references: references,
                options: new CSharpCompilationOptions(
                    OutputKind.DynamicallyLinkedLibrary,
                    optimizationLevel: OptimizationLevel.Release));

            using (var ms = new MemoryStream())
            {
                EmitResult result = compilation.Emit(ms);

                if (!result.Success)
                {
                    IEnumerable<Diagnostic> failures = result.Diagnostics.Where(diagnostic =>
                        diagnostic.IsWarningAsError ||
                        diagnostic.Severity == DiagnosticSeverity.Error).ToList();

                    foreach (Diagnostic diagnostic in failures)
                    {
                        errors.Add($"{diagnostic.Id}: {diagnostic.GetMessage()}");
                    }
                }
                else
                {
                    ms.Seek(0, SeekOrigin.Begin);

                    Assembly assembly = AssemblyLoadContext.Default.LoadFromStream(ms);
                    //support for Namespace , as well as backward compatibility for existing functions
                    var type = assembly.GetTypes().FirstOrDefault(x => x.Name.EndsWith("FissionFunction"));
                    var info = type.GetMember("Execute").First() as MethodInfo;
                    return new Function(assembly, type, info);
                }
            }
            return null;
        }

        public Function Compilev2(string code, out List<string> errors, out List<string>  oinfo)
        {
            errors = new List<string>();
            oinfo = new List<string>();

            #region syntext tree and default reference build
            SyntaxTree syntaxTree = CSharpSyntaxTree.ParseText(code);

            string assemblyName = Path.GetRandomFileName();

            var coreDir = Directory.GetParent(typeof(Enumerable).GetTypeInfo().Assembly.Location);

            Console.WriteLine("Adding core references !!");
            List<MetadataReference> references = new List<MetadataReference>
            {
                MetadataReference.CreateFromFile(coreDir.FullName + Path.DirectorySeparatorChar + "mscorlib.dll"),
                MetadataReference.CreateFromFile(coreDir.FullName + Path.DirectorySeparatorChar + "netstandard.dll"),
                MetadataReference.CreateFromFile(typeof(object).GetTypeInfo().Assembly.Location),
                MetadataReference.CreateFromFile(Assembly.GetEntryAssembly().Location),
                MetadataReference.CreateFromFile(typeof(System.Runtime.Serialization.Json.DataContractJsonSerializer).GetTypeInfo().Assembly.Location)
            };

            Console.WriteLine("Adding parent assembly based references !!");
            foreach (var referencedAssembly in Assembly.GetEntryAssembly().GetReferencedAssemblies())
            {
                var assembly = Assembly.Load(referencedAssembly);
                references.Add(MetadataReference.CreateFromFile(assembly.Location));               
            }

            #endregion

            #region load function specs based dlls

            Console.WriteLine($"going to get function specification...");
            //load all available dlls from  deployment folder in dllinfo object
            functionSpecification = EnvironmentHelper.Instance.GetFunctionSpecs(packagepath);

            Console.WriteLine($"going to get package dlls...");
            //iterate and all all libraries mentioned 
            foreach (var library in functionSpecification.libraries)
            {
                string dllCompletePath = Path.Combine(packagepath, library.path).GetrelevantPathAsPerOS();
                references.Add(MetadataReference.CreateFromFile(dllCompletePath));
                Console.WriteLine($"referred folder based dll : {dllCompletePath} from package {library.nugetPackage}");
            }
            Console.WriteLine($"referred all available dlls!!");
            oinfo.Add("referred all available dlls!!");

            #endregion 

            #region dynamic resolve handler registration 
            AppDomain currentDomain = AppDomain.CurrentDomain;
            currentDomain.AssemblyResolve += CurrentDomain_AssemblyResolve;

            #endregion

            #region function compile

            Console.WriteLine($"Trying to Compile");
            CSharpCompilation compilation = CSharpCompilation.Create(
                assemblyName,
                syntaxTrees: new[] { syntaxTree },
                references: references,
                options: new CSharpCompilationOptions(
                    OutputKind.DynamicallyLinkedLibrary,
                    optimizationLevel: OptimizationLevel.Release));

            using (var ms = new MemoryStream())
            {
                EmitResult result = compilation.Emit(ms);

                if (!result.Success)
                {
                    Console.WriteLine($"Compile Failed , see pod logs for more details");
                    IEnumerable<Diagnostic> failures = result.Diagnostics.Where(diagnostic =>
                        diagnostic.IsWarningAsError ||
                        diagnostic.Severity == DiagnosticSeverity.Error).ToList();

                    foreach (Diagnostic diagnostic in failures)
                    {
                        errors.Add($"{diagnostic.Id}: {diagnostic.GetMessage()}");
                        Console.WriteLine($"COMPILE ERROR :{diagnostic.Id}: {diagnostic.GetMessage()}", "ERROR");
                    }
                }
                else
                {
                    oinfo.Add("COMPILE SUCCESS!!");
                    Console.WriteLine($"COMPILE SUCCESS!!");

                    ms.Seek(0, SeekOrigin.Begin);

                    Assembly assembly = AssemblyLoadContext.Default.LoadFromStream(ms);
                    //var type = assembly.GetType("FissionFunction");
                    //support for Namespace , as well as backward compatibility for existing functions
                    var type = assembly.GetTypes().FirstOrDefault(x => x.Name.EndsWith("FissionFunction"));
                    //assembly.GetTypes().Where(x=>x.Name.ToLower().EndsWith("FissionFunction".ToLower())).FirstOrDefault();
                    var info = type.GetMember("Execute").First() as MethodInfo;
                    return new Function(assembly, type, info);
                }
            }
            return null;

            #endregion
        }

        private Assembly CurrentDomain_AssemblyResolve(object sender, ResolveEventArgs args)
        {
            //This handler is called only when the common language runtime tries to bind to the assembly and fails.

            Console.WriteLine($"Dynamically trying to load dll {(args.Name.Substring(0, args.Name.IndexOf(",")).ToString() + ".dll").ToLower()} in parent assembaly");

            //Retrieve the list of referenced assemblies in an array of AssemblyName.
            Assembly MyAssembly = null, objExecutingAssemblies;
            string strTempAssmbPath_relative = "", strTempAssmbPath_absolute = "";

            objExecutingAssemblies = Assembly.GetExecutingAssembly();
            AssemblyName[] arrReferencedAssmbNames = objExecutingAssemblies.GetReferencedAssemblies();

            ////Loop through the array of referenced assembly names.

            //load all available dlls from  deployment folder in dllinfo object         
            if (functionSpecification.libraries.Any(x => x.name.ToLower() == (args.Name.Substring(0, args.Name.IndexOf(",")).ToString() + ".dll").ToLower()))
            {
                strTempAssmbPath_relative = functionSpecification.libraries.Where(x => x.name.ToLower() == (args.Name.Substring(0, args.Name.IndexOf(",")).ToString() + ".dll").ToLower()).FirstOrDefault().path;
                strTempAssmbPath_absolute = Path.Combine(packagepath, strTempAssmbPath_relative);
                Console.WriteLine($"loading dll in parent assembly :{strTempAssmbPath_absolute.GetrelevantPathAsPerOS()}");
                //Load the assembly from the specified path. 
                MyAssembly = Assembly.LoadFile(strTempAssmbPath_absolute.GetrelevantPathAsPerOS());
                Console.WriteLine($"Load success for :{strTempAssmbPath_absolute.GetrelevantPathAsPerOS()}");
            }

            if (MyAssembly == null)
            {
                Console.WriteLine($"WARNING !!! unabel to locate dll  :{(args.Name.Substring(0, args.Name.IndexOf(",")).ToString() + ".dll").ToLower()} ", "WARNING");
            }
            //Return the loaded assembly.
            return MyAssembly;
        }
    }
}
