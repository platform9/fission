using Builder.Utility;
using NugetWorker;
using System;
using System.IO;
using Builder.Engine;

namespace Builder
{
    class Builder
    {
        static void Main(string[] args)
        {
            Console.WriteLine("Builder Task Begins!");
            //create log file name for this session 
            var logFileName = $"{DateTime.Now.ToString("yyyy_MM_dd")}_{Guid.NewGuid().ToString()}.log";
            Console.WriteLine($"going to create logger!!");
                        
            try
            {
                string _logdirectory = BuilderHelper.Instance.builderSettings.BuildLogDirectory;
                BuilderHelper.Instance._logFileName = Path.Combine(_logdirectory, logFileName);
                BuilderHelper.Instance.logger = new Utility.Logger(
                           BuilderHelper.Instance._logFileName);

                //set the same for nuget engine dll
                NugetHelper.Instance.logger = new NugetWorker.Logger(BuilderHelper.Instance._logFileName);

                Console.WriteLine($"detailed logs for this build will be at : {Path.Combine(_logdirectory, logFileName)}!!");


                BuilderEngine builderEngine = new BuilderEngine();
                builderEngine.BuildPackage().Wait();
            }
            catch (Exception ex)
            {
                string detailedException = string.Empty;
                try
                {
                    detailedException= BuilderHelper.Instance.DeepException(ex);
                    Console.WriteLine($"Exception During Build : {Environment.NewLine} {ex.Message} | {ex.StackTrace} | {Environment.NewLine} {detailedException}");
                }
                catch(Exception childEx)
                {
                    //do nothing , just log original exception
                    Console.WriteLine($"{Environment.NewLine} Exception During Build :{ex.Message} |{Environment.NewLine}  {ex.StackTrace} {Environment.NewLine} ");
                }              

                //now throw back exception so that build gets failed via builder 
                throw;
            }

            Console.WriteLine("Builder Task Ends!");
        }
    }
}
