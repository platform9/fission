# frozen_string_literal: true

require 'benchmark'
require 'json'

module Fission
  module Specializer
    
    CODE_PATH = '/userfunc/user'

    def self.call(env)
      request = Request.new(env)

      request.logger.info("Codepath defaulting to #{CODE_PATH}")
      time = Benchmark.measure { load CODE_PATH }
      request.logger.info("User code loaded in #{(time.real * 1000).round(3)}ms")

      # set to "handler" for v1 specialization
      $handler = method(:handler)

      Rack::Response.new([], 201).finish
    rescue => e
      request.logger.error(%(Specialization failed - #{e}\n#{e.backtrace.join("\n")}))
      Rack::Response.new(['500 Internal Server Error'], 500, {}).finish
    end
  end

  module V2
    module Specializer

      def self.load_vendor(path)
        gems = Dir[File.join(path, 'vendor/bundle/ruby/*/gems/*/lib')]
        exts = Dir[File.join(path, 'vendor/bundle/ruby/*/extensions/x86_64-linux/*/*')]

        $LOAD_PATH.unshift(*gems)
        $LOAD_PATH.unshift(*exts)
      end

      def self.call(env)
        request = Request.new(env)

        body = JSON.parse(request.body.read)
        path = body['filepath']
        func = body['functionName']

        time = Benchmark.measure do
          if File.file?(path)
            # If pointing to just a single file, all we need to do is load it
            request.logger.debug("Loading file #{path}")

            load path
          elsif File.directory?(path)
            # First we want to load all the vendor files
            load_vendor(path)

            # We then want to get all the .rb files in this src
            rb_files = File.join(path, "**/*.rb")

            # But we don't want to include the vendor files that we loaded above
            vendor_dir = File.join(path, 'vendor')
            src_files = Dir[rb_files].reject {|d| d.start_with?(vendor_dir) }

            request.logger.debug("Loading sources #{src_files}")

            src_files.each do |file|
              load file
            end
          else
            request.logger.error(%(Specialization failed - could not find src at #{path}}))
            Rack::Response.new(['500 Internal Server Error'], 500, {}).finish

            return
          end
        end
        request.logger.info("User code loaded in #{(time.real * 1000).round(3)}ms")

        # set global handler for this specialization
        $handler = method(func)

        Rack::Response.new([], 201).finish
      end
    end
  end
end
