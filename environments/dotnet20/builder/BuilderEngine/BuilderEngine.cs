using Builder.Model;
using Builder.Utility;
using NugetWorker;
using System;
using System.Collections.Generic;
using System.Text;
using System.Threading.Tasks;
using System.Linq;
using NuGet.Packaging;
using System.IO;
using Microsoft.CodeAnalysis;
using Microsoft.CodeAnalysis.CSharp;
using System.Reflection;
using Microsoft.CodeAnalysis.Emit;
using System.Runtime.Loader;
using Newtonsoft.Json;

namespace Builder.Engine
{
   public class BuilderEngine
    {
        public string SRC_PKG = string.Empty;
        public string DEPLOY_PKG = string.Empty;

        List<DllInfo> dllInfos = new List<DllInfo>();
        List<ExcludeDll> excludeDlls = new List<ExcludeDll>();
        List<IncludeNuget> includeNugets = new List<IncludeNuget>();
        List<string> compile_errors = new List<string>();
        List<string> compile_info = new List<string>();

        public BuilderEngine()
        {
            SRC_PKG = Environment.GetEnvironmentVariable("SRC_PKG");
            DEPLOY_PKG = Environment.GetEnvironmentVariable("DEPLOY_PKG");
        }
    

        public async Task BuildPackage()
        {
            await BuildDllInfo();

            Console.WriteLine("DLL Info Gathered!!");
            // try to compile the function and if compilation succeed ,then create func spec file
            //this enables us to find compilation issues during package creation itself thus saving time 
            // however this feature impose that  the function file name should be func.cs 
            //if we don't want it , we can comment the  TryCompile() logic
            Console.WriteLine("Trying to compile it during build itself !!");
            bool compiled =await TryCompile();
            Console.WriteLine($"Compilation result Gathered as : {compiled}!!");
            if (compiled)
            {

                //nowwhatever has been done so far and all files which are generated are in /app folder where dll resides
                //thus copy relevant thing in SRC_PKG as it is ,but lest skip it here , we shall do it in build.sh
                CopyToSourceDir();
                Console.WriteLine($"Copy to Source Done!!");
                //build the function specs
                await BuildSpecs();
                Console.WriteLine("BuildSpecs Done!!");
            }
            else
            {
                Console.WriteLine("Compilation failed , throwing exception !!");
                   foreach(var error  in  compile_errors)
                    {
                        Console.WriteLine($"COMPILATION ERROR : {error}");
                    }
                throw new Exception($"COMPILATION FAILED !! , See builder logs for details , total Errors :  {compile_errors.Count}");
                
            }

        }

        public void CopyToSourceDir()
        {
            //to create folder if it doesnt already exists
            string destinationFile=Path.Combine(SRC_PKG, BuilderHelper.Instance.builderSettings.DllDirectory, "dummy.txt");
            new FileInfo(destinationFile).Directory.Create();
           

            //copy all dlls
            foreach (var dllinfo in dllInfos)
            {
                string filename = Path.GetFileName(dllinfo.path);
                destinationFile = Path.Combine(SRC_PKG, BuilderHelper.Instance.builderSettings.DllDirectory, filename);
                File.Copy(dllinfo.path, destinationFile,true);
            }


            //copy logs , well there is not point as logs are still being generated

            //create dir if not exist
            //new FileInfo(Path.Combine(SRC_PKG, BuilderHelper.Instance._logFileName)).Directory.Create();
            
            //File.Copy(BuilderHelper.Instance._logFileName, Path.Combine(SRC_PKG, BuilderHelper.Instance._logFileName));
            //BuilderHelper.Instance.logger.Log($"All Required Files copied to {SRC_PKG}");
        }
        
        public async Task<bool> TryCompile()
        {
            bool isSuccess = false;

            string CODE_PATH = Path.Combine(SRC_PKG, BuilderHelper.Instance.builderSettings.functionBodyFileName);
            if (!File.Exists(CODE_PATH))
            {
                Console.WriteLine($"Source Code not found at : {CODE_PATH} !" +
                    $" to use TryCompile() in Builder, make sure , your main function file name is " +
                    $"{BuilderHelper.Instance.builderSettings.functionBodyFileName} and " +
                    $"it is located at root of zip!!" );
                return isSuccess;
            }

            var code = File.ReadAllText(CODE_PATH);
            isSuccess = await Compile(code);

            return isSuccess;
        }

        public async Task<bool> Compile(string code)
        {
            bool isSuccess = false;

            #region assembly init and parent dll references
            SyntaxTree syntaxTree = CSharpSyntaxTree.ParseText(code);
            string assemblyName = Path.GetRandomFileName();

            var coreDir = Directory.GetParent(typeof(Enumerable).GetTypeInfo().Assembly.Location);

            List<MetadataReference> references = new List<MetadataReference>
            {
                MetadataReference.CreateFromFile(coreDir.FullName + Path.DirectorySeparatorChar + "mscorlib.dll"),
                MetadataReference.CreateFromFile(coreDir.FullName + Path.DirectorySeparatorChar + "netstandard.dll"),
                MetadataReference.CreateFromFile(typeof(object).GetTypeInfo().Assembly.Location),
                MetadataReference.CreateFromFile(Assembly.GetEntryAssembly().Location),
                MetadataReference.CreateFromFile(typeof(System.Runtime.Serialization.Json.DataContractJsonSerializer).GetTypeInfo().Assembly.Location)
            };

            foreach (var referencedAssembly in Assembly.GetEntryAssembly().GetReferencedAssemblies())
            {
                var assembly = Assembly.Load(referencedAssembly);
                references.Add(MetadataReference.CreateFromFile(assembly.Location));
                BuilderHelper.Instance.logger.Log($"Refering assembly based dls :  {assembly.Location}");
            }

            #endregion

            #region handler registration for runtime resolution
            //now add handler for missing dlls for parent app domain as same assemblies should be needed 
            //for parent , thus refering from https://support.microsoft.com/en-in/help/837908/how-to-load-an-assembly-at-runtime-that-is-located-in-a-folder-that-is
            AppDomain currentDomain = AppDomain.CurrentDomain;
            currentDomain.AssemblyResolve += CurrentDomain_AssemblyResolve;

            #endregion

            BuilderHelper.Instance.logger.Log($"dynamic handlar registered!!");
            #region nuget dll reference add
            //now add those dll reference
            foreach (var dll in dllInfos)
            {
                BuilderHelper.Instance.logger.Log($"refering nuget based dll : {dll.path}");
                references.Add(MetadataReference.CreateFromFile(dll.path));
            }

            #endregion

            #region compilation 

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
                    BuilderHelper.Instance.logger.Log($"Compile Failed!!!!",true);
                    IEnumerable<Diagnostic> failures = result.Diagnostics.Where(diagnostic =>
                        diagnostic.IsWarningAsError ||
                        diagnostic.Severity == DiagnosticSeverity.Error).ToList();

                    foreach (Diagnostic diagnostic in failures)
                    {
                        compile_errors.Add($"{diagnostic.Id}: {diagnostic.GetMessage()}");
                        BuilderHelper.Instance.logger.Log($"COMPILE ERROR :{diagnostic.Id}: {diagnostic.GetMessage()}");
                    }
                }
                else
                {
                    BuilderHelper.Instance.logger.Log("Compile Success!!",true);
                    isSuccess = true;
                }
            }

            #endregion 
            return isSuccess;

        }

        public async Task BuildSpecs()
        {
            //create function specs C# object
            FunctionSpecification functionSpecification = new FunctionSpecification();

            functionSpecification.functionName = BuilderHelper.Instance.builderSettings.functionBodyFileName;

            foreach (var dllinfo in dllInfos)
            {
                //here is the tweak , as this path is based on execution directoy , thus choose the relative path
                string destinationFile = Path.Combine(BuilderHelper.Instance.builderSettings.DllDirectory, Path.GetFileName(dllinfo.path)).GetrelevantPathAsPerOS();

                Library library = new Library()
                {
                    name=dllinfo.name,
                    nugetPackage=dllinfo.rootPackage,
                    path = destinationFile
                };
                functionSpecification.libraries.Add(library);
            }

            //serialize that object to save it in json file
            string funcMetaJson= JsonConvert.SerializeObject(functionSpecification);

            string funcMetaFile = Path.Combine(this.SRC_PKG, BuilderHelper.Instance.builderSettings.functionSpecFileName);
            BuilderHelper.Instance.WriteTofile(funcMetaFile, funcMetaJson);

        }

        public async Task BuildDllInfo()
        {
            //read the nuget file and download nuget packages
            includeNugets = BuilderHelper.Instance.GetNugettoInclude(SRC_PKG);
            //set the nuget logger to same logger


            foreach (var nuget in includeNugets)
            {
                NugetEngine nugetEngine = new NugetEngine();
                await nugetEngine.GetPackage(nuget.packageName, nuget.version);

                //add the list of dlls received via this package in master list
                dllInfos.AddRange(nugetEngine.dllInfos);
            }

            //now do a distinct of all dlls paths as multiple packaged might have added same dll
            dllInfos = dllInfos.DistinctBy(x => x.path).ToList();

#if DEBUG
            dllInfos.LogDllPathstoCSV("preFilter.CSV");
#endif

            //exclude the dlls from exclude file
            excludeDlls = BuilderHelper.Instance.GetDllstoExclude(SRC_PKG);
            foreach (var excludedll in excludeDlls)
            {
                BuilderHelper.Instance.logger.Log($"trying to remove , if available : {excludedll.dllName} from package  {excludedll.packageName}");
                dllInfos.RemoveAll(x => x.rootPackage.ToLower() == excludedll.packageName.ToLower() && x.name.ToLower() == excludedll.dllName.ToLower());
            }

#if DEBUG
            //log dlls in debug mode
            dllInfos.LogDllPathstoCSV("PostFilter.CSV");
#endif

        }

        private Assembly CurrentDomain_AssemblyResolve(object sender, ResolveEventArgs args)
        {
            //This handler is called only when the common language runtime tries to bind to the assembly and fails.

            BuilderHelper.Instance.logger.Log($"Dynamically trying to load dll {(args.Name.Substring(0, args.Name.IndexOf(",")).ToString() + ".dll").ToLower()} in parent assembaly");

            //Retrieve the list of referenced assemblies in an array of AssemblyName.
            Assembly MyAssembly = null, objExecutingAssemblies;
            string strTempAssmbPath = "";

            objExecutingAssemblies = Assembly.GetExecutingAssembly();
            AssemblyName[] arrReferencedAssmbNames = objExecutingAssemblies.GetReferencedAssemblies();

            ////Loop through the array of referenced assembly names.

            if (dllInfos.Any(x => x.name.ToLower() == (args.Name.Substring(0, args.Name.IndexOf(",")).ToString() + ".dll").ToLower()))
            {
                strTempAssmbPath = dllInfos.Where(x => x.name.ToLower() == (args.Name.Substring(0, args.Name.IndexOf(",")).ToString() + ".dll").ToLower()).FirstOrDefault().path;

                BuilderHelper.Instance.logger.Log($"loading dll in parent :{strTempAssmbPath}");

                //Load the assembly from the specified path. 
                MyAssembly = Assembly.LoadFile(strTempAssmbPath);
            }

            if (MyAssembly == null)
            {
                BuilderHelper.Instance.logger.Log($"WARNING !!! unabel to locate dll  :{(args.Name.Substring(0, args.Name.IndexOf(",")).ToString() + ".dll").ToLower()} ", true);
            }
            //Return the loaded assembly.
            return MyAssembly;
        }

    }
}
