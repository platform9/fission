# Fission: dotnet 2.0 C# Environment Builder

This is a simple dotnet core 2.0 C# environment builder for Fission.

It's a docker image containing the dotnet 2.0.0 (core) run-time builder. This image read the source package and uses 
*roslyn* to compile the source package code and creates deployment package out of it.
This enables using nuget packages as part of function and thus user can use extended functionality in fission functions via nuget.

During build , builder also does a pre-compile to prevent any compilation issues during function environment pod specialization.
Thus we get the function compilation issues during builder phase in package info's build logs itself.

**Note** : In future we can further enhance the compiled assembly to be saved as physical file in deployment package , 
as this will save cold start time for function. 

Now , once after the build is finished, the output package (deploy archive) will be uploaded to storagesvc to store.
Then, during the specialization, the fetcher inside function pod will fetch the package from storagesvc for function loading
 and will call on the  **/v2/specialized** endpoint of fission environment with required parameters.

There further environment will compile it and execute the function.


Example of simplest possible class to be executed:

The source package structure in zip file :

```
 Source Package zip :
 --source.zip
	|--func.cs
	|--nuget.txt
	|--exclude.txt
	|--....MiscFiles(optional)
	|--....MiscFiles(optional)
```

**func.cs** --> This contains original function body with Executing method name as : Execute

 
**nuget.txt**--> this file contains list of nuget packages required by your function , in this file put one line per nuget with nugetpackage name:version(optional) format, for example :

```
RestSharp
CsvHelper
Newtonsoft.json:10.2.1.0
```


 this should match the following regex as mentioned in builderSetting.json
```
"NugetPackageRegEx": "\\:?\\s*(?<package>[^:\\n]*)(?:\\:)?(?<version>.*)?"
```

  
 **exclude.txt**--> as nuget.txt will download original package and their dependent packages , thus sometime dependent packages might not be
that useful and can break compilation , thus this file contains list of dlls of specific nuget packages which doesn't need to be  added during compilation if  they  break compilation .Put one line per nuget with dllname:nugetpackagename formate ,for example :
 
```
Newtonsoft.json:Newtonsoft.json.dll
```
this should match the following regex as mentions in builderSetting.json


```
"ExcludeDllRegEx": "\\:?\\s*(?<package>[^:\\n]*)(?:\\:)?(?<dll>.*)?",
```
 From above , builder will create a deployment package with all dlls in a folder and one function specification file :
 Deployment Package zip :

```
 --Deploye.zip
	|--func.cs
	|--nuget.txt
	|--exclude.txt
	|--dll()
		|--newtonsoft.json.dll
		|--restsharp.dll
		|--csvhelper.dll
	|--logs()
		|-->logFileName
	|--func.meta.json // this is the function specific file
	|--....MiscFiles(optional)
	|--....MiscFiles(optional)
```
Here are commands and detailed example for the same .

lets say my source package zip name is *funccsv.zip* :

**Content of func.cs:**
```
using System;
using Fission.DotNetCore.Api;

public class FissionFunction 
{
    public string Execute(FissionContext context){
		string res="initial value";
	        try
            {
				context.Logger.WriteInfo("Staring..... ");
				res=$" sample object by getting Enum of   CsvHelper nuget dll: { CsvHelper.Caches.NamedIndex.ToString()}";
            }  
            catch(Exception ex)
            {
				context.Logger.WriteError(ex.Message);
                res = ex.Message;
            }
		context.Logger.WriteInfo("Done!");
		return res;
    }
}
```

**Content of  *nuget.txt***
```
CsvHelper
```
**Content of exclude.txt**

As we don't want to exclude any specific dll thus we shall leave it as empty.

Now check name of existing environments & functions as we want to create a unique environment for this .Net Core if not already present

```
fission env list
fission fn list
 ```
 Create Environment with builder (choose a unique which doesn't exist , here we have chosen : dotnetcorewithnuget  ) 
 also suppose the builder image name is fissiondotnet20-builder and hosted on Docker Hub as fission/dotnet20-builder
 ```
fission environment create --name dotnetcorewithnuget --image fission/dotnet20-env  --builder  fission/dotnet20-builder
 ```
 Verify fission-builder and fission-function namespace for new pods (pods name beginning with env name which we have given like *dotnetcorewithnuget-xxx-xxx*)
 ```
kubectl get pods -n fission-builder
kubectl get pods -n fission-function
 ```
Create Package from source zip using this environment name , this will output some package name created..
 ```
fission package create --src funccsv.zip --env dotnetcorewithnuget
 ```
 Note down output package name lets say it is *funccsv-zip-xyz* now check its status using package info command , this will give the status
 on what happened with builder and test compilation in builder.
 
 ```
fission package info --name funccsv-zip-xyz
```

#Status of package should be f*ailed / running / succeeded* .
 Wait if the status is running , until it fails or succeeded. For detailed build logs, you can shell into builder pod in fission-builder namespace and verify log location mentioned in above command's result output.

**Note** : Even If the result is succeeded , please have a look at detailed build logs to see compilation success and builder job done.

Now If the result is succeeded , then go ahead and create function using this package.

*--entrypoint* flag is optional if your function body file name is func.cs  (which it should be as builder need that), else put the filename (without extension )
 ```
 fission fn create --name dotnetcsvtest --pkg funccsv-zip-xyz --env dotnetcorewithnuget --entrypoint "func"
 ```
Test the function execution :

``` 
 fission fn test --name dotnetcsvtest
```
above would execute the function and will output the enum value as written in dll.
rest of the feature are same as normal fission environment.

**Benefit of using builder** :

1. Ability to use various nuget packages.
2. Ability to use many additional files and functions as part of deployment package.
3. Ability to know the compilation issue in advance via package logs , instead of environment giving compilation issue.
4. Reusability of same deployment package.



