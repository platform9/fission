using System;
using System.Collections.Generic;
using System.IO;
using System.Security.Cryptography.X509Certificates;
using System.Text;
using Nancy;
using Newtonsoft.Json;


namespace Fission.DotNetCore.Api
{
    public class FissionContext
    {
        public FissionContext(Dictionary<string, object> args, Logger logger, FissionHttpRequest request)
        {
            if (args == null) throw new ArgumentNullException(nameof(args));
            if (logger == null) throw new ArgumentNullException(nameof(logger));
            if (request == null) throw new ArgumentNullException(nameof(request));
            Arguments = args;
            Logger = logger;
            Request = request;
        }

        public Dictionary<string, object> Arguments { get; private set; }

        public FissionHttpRequest Request { get; private set; }

        public Logger Logger { get; private set; }

        public static FissionContext Build(Request request, Logger logger)
        {
            return new FissionContext(((DynamicDictionary)request.Query).ToDictionary(),
                                        logger,
                                        new FissionHttpRequest(request));
        }

        //this is a dummy, not being implemented, just to pass compilation
        //actual execution is written in environment to use the app settings as there we need it
        public T GetSettings<T>(string relativePath)
        {
            //intentionally doing it as these are just dummy methods not being called
            //but if tomorrow if we decide to give implementation for execution in build then we 
            //need to implement it
            throw new NotImplementedException();
        }

        //this is a dummy, not being implemented, just to pass compilation
        //actual execution is written in environment to use the app settings as there we need it
        private string GetSettingsJson(string relativePath)
        {
            //intentionally doing it as these are just dummy methods not being called
            //but tomorrow if we decide to give implementation for execution in build then we 
            //need to implement it
            throw new NotImplementedException();
        }

    }

    public class Logger
    {
        public void Write(Severity severity, string format, params object[] args)
        {
            Console.WriteLine($"{DateTime.Now.ToString("MM/dd/yy H:mm:ss zzz")} {severity}: " + format, args);
        }

        public void WriteInfo(string format, params object[] args)
        {
            Write(Severity.Info, format, args);
        }

        public void WriteWarning(string format, params object[] args)
        {
            Write(Severity.Warning, format, args);
        }

        public void WriteError(string format, params object[] args)
        {
            Write(Severity.Error, format, args);
        }

        public void WriteCritical(string format, params object[] args)
        {
            Write(Severity.Critical, format, args);
        }

        public void WriteVerbose(string format, params object[] args)
        {
            Write(Severity.Verbose, format, args);
        }
    }

    public enum Severity
    {
        Info,
        Warning,
        Error,
        Critical,
        Verbose
    }

    public class FissionHttpRequest
    {
        private readonly Request _request;
        internal FissionHttpRequest(Request request)
        {
            if (request == null) throw new ArgumentNullException(nameof(request));
            _request = request;
        }

        public Stream Body { get { return _request.Body; } }

        public string BodyAsString()
        {
            int length = (int)_request.Body.Length;
            byte[] data = new byte[length];
            _request.Body.Read(data, 0, length);
            return Encoding.UTF8.GetString(data);
        }
        
        public Dictionary<string, IEnumerable<string>> Headers
        {
            get
            {
                var headers = new Dictionary<string, IEnumerable<string>>();
                foreach (var kv in _request.Headers)
                {
                    headers.Add(kv.Key, kv.Value);
                }
                return headers;
            }
        }

        public X509Certificate Certificate { get { return _request.ClientCertificate; } }
        public string Url { get { return _request.Url.ToString(); } }
        public string Method { get { return _request.Method; } }
    }
}
