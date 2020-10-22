using Fission.DotNetCore.Compiler;
using Fission.DotNetCore.Api;
using System.Collections.Generic;
using Nancy;
using System.IO;
using System;
using Nancy.IO;
using Fission.DotNetCore.Utilty;
using Fission.DotNetCore.Model;
using Nancy.Extensions;

namespace Fission.DotNetCore
{
    public class ExecutorModule : NancyModule
    {
        private static string PackagePath = string.Empty;
#if DEBUG
        private const string CODE_PATH = "/tmp/func.cs";
#else
        private const string CODE_PATH = "/userfunc/user";
#endif
        private static Function _userFunc;
        private static Logger _logger = new Logger();

        public ExecutorModule()
        {
            Post("/specialize", args => Specialize());
            Post("/v2/specialize", args => Specializev2());
            Get("/", _ => Run());
            Post("/", _ => Run());
            Put("/", _ => Run());
            Head("/", _ => Run());
            Options("/", _ => Run());
            Delete("/", _ => Run());
        }

        private object Specializev2()
        {
            Console.WriteLine("Call Reached at  /v2/specialize");

            try
            {
                var errors = new List<string>();
                var oinfo = new List<string>();
                var _request = Request;
                var _body = Request.Body;
                // Request.Body.Position = 0; use it only if request has already been read before that
                var _requestBodystring = RequestStream.FromStream(Request.Body).AsString();
                Console.WriteLine($"Request received by endpoint from builder : {_requestBodystring}");
                BuilderRequest builderRequest = EnvironmentHelper.Instance.GetBuilderRequest(_requestBodystring);
                if (builderRequest == null)
                {
                    Console.WriteLine("Error : Unable to parse builder request!!");
                    throw new Exception("Error : Unable to parse builder request!!");
                }

                string functionPath = string.Empty;
                // functionPath = Path.Combine(builderRequest.filepath, $"{builderRequest.functionName}.cs");

                PackagePath = builderRequest.filepath;
                //following will enable us to skip --entrypoint flag during function creation 
                if (!string.IsNullOrWhiteSpace(builderRequest.functionName))
                {
                    functionPath = Path.Combine(builderRequest.filepath, $"{builderRequest.functionName}.cs");
                }
                else
                {
                    functionPath = Path.Combine(builderRequest.filepath, EnvironmentHelper.Instance.environmentSettings.functionBodyFileName);
                }

                Console.WriteLine($"Going to read function body from path : {functionPath}");

                if (File.Exists(functionPath))
                {
                    var code = File.ReadAllText(functionPath);
                    try
                    {
                        FissionCompiler fissionCompiler = new FissionCompiler(builderRequest.filepath);
                        _userFunc = fissionCompiler.Compilev2(code, out errors, out oinfo);
                    }
                    catch (Exception ex)
                    {
                        Console.WriteLine($"Error getting _userFunc :{ex.Message} , Trace : {ex.StackTrace}");
                    }
                    if (_userFunc == null)
                    {
                        var errstr = string.Join(Environment.NewLine, errors);
                        _logger.WriteError(errstr);
                        Console.WriteLine($"Error  _userFunc is null :{errstr}");
                        var response = (Response)errstr;
                        response.StatusCode = HttpStatusCode.InternalServerError;
                        return response;

                    }
                    else
                    {
                        //try to retrun few details 
                        var infostr = string.Join(Environment.NewLine, oinfo);
                        _logger.WriteInfo(infostr);
                        var response = (Response)infostr;
                        response.StatusCode = HttpStatusCode.OK;
                        return response;
                    }
                }
                else
                {
                    var errstr = $"Unable to locate code at '{functionPath}'";
                    _logger.WriteError(errstr);
                    var response = (Response)errstr;
                    response.StatusCode = HttpStatusCode.InternalServerError;
                    return response;
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Exception occurred {ex.Message} | {ex.StackTrace}");
                var errstr = $"Exception occurred {ex.Message} | {ex.StackTrace}";
                _logger.WriteError(errstr);
                var response = (Response)errstr;
                response.StatusCode = HttpStatusCode.InternalServerError;
                return response;
            }
        }

        private object Specialize()
        {
            var errors = new List<string>();
            if (File.Exists(CODE_PATH))
            {
                var code = File.ReadAllText(CODE_PATH);
                _userFunc = FissionCompiler.Compile(code, out errors);
                if (_userFunc == null)
                {
                    var errstr = string.Join(Environment.NewLine, errors);
                    _logger.WriteError(errstr);
                    var response = (Response)errstr;
                    response.StatusCode = HttpStatusCode.InternalServerError;
                    return response;

                }
                return null;
            }
            else
            {
                var errstr = $"Unable to locate code at '{CODE_PATH}'";
                _logger.WriteError(errstr);
                var response = (Response)errstr;
                response.StatusCode = HttpStatusCode.InternalServerError;
                return response;
            }
        }

        private object Run()
        {
            if (_userFunc == null)
            {
                var response = (Response)"Generic container: no requests supported";
                response.StatusCode = HttpStatusCode.InternalServerError;
                return response;
            }
            
            try
            {
                var context = FissionContext.Build(Request, new Logger());
                //set the package path ,as that will be required to get appsetting files from package
                context.PackagePath = PackagePath;
                return _userFunc.Invoke(context);
            }
            catch (Exception e)
            {
                _logger.WriteError(e.ToString());
                var response = (Response)e.Message;
                response.StatusCode = HttpStatusCode.BadRequest;
                return response;
            }
        }
    }
}

