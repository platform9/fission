# frozen_string_literal: true

require_relative 'context'

module Fission
  module Handler
    def self.call(env)
      context = Context.new(env)

      response = if $handler.arity > 0
        $handler.call(context)
      else
        $handler.call
      end

      response.is_a?(Array) ? response : Rack::Response.new([response]).finish

    rescue => e
      context.logger.error(%(Function failed - #{e}\n#{e.backtrace.join("\n")}))
      Rack::Response.new(['500 Internal Server Error'], 500, {}).finish
    end
  end
end
