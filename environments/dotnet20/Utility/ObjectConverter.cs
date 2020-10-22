using Fission.DotNetCore.Model;
using Newtonsoft.Json;
using System;
using System.Collections.Generic;
using System.Text;

namespace Fission.DotNetCore.Utilty
{
    public sealed class ObjectConverter
    {
        private static readonly Lazy<ObjectConverter> lazy =
         new Lazy<ObjectConverter>(() => new ObjectConverter());

        public static ObjectConverter Instance { get { return lazy.Value; } }

        private ObjectConverter() {}

        public EnvironmentSettings GetWatcherSettingsFromJson(string json)
        {
            return JsonConvert.DeserializeObject<EnvironmentSettings>(json);
        }

        public FunctionSpecification GetFunctionSpecificationFromJson(string json)
        {
            return JsonConvert.DeserializeObject<FunctionSpecification>(json);
        }
    }
}
