# frozen_string_literal: true

require 'rack'
require 'thin'
require 'logger'

require_relative 'fission/specializer'
require_relative 'fission/handler'

$handler = nil

app = Rack::Builder.new do
  use Rack::Logger, Logger::DEBUG
  use Rack::CommonLogger

  map "/specialize" do
    run Fission::Specializer
  end

  map '/v2/specialize' do
    run Fission::V2::Specializer
  end

  map "/healthz" do
    run ->(env) { [ 200, {}, [] ] }
  end

  map '/' do
    run Fission::Handler
  end
end

Rack::Handler::Thin.run app, Host: '0.0.0.0', Port: 8888
