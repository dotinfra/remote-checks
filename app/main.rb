require 'rubygems'
require 'sinatra/base'
require 'benchmark'
require 'json'
require 'net/http'
require 'net/https'

class NetChecks < Sinatra::Base

# If try to access to root, do redirect => skillstar.com
# HTTP Response : 302
  get '/' do
    redirect 'http://www.skillstar.com'
  end

# Getting JSON serialized return of checks
  get '/checks' do
    content_type :json

    url = "#{params[:url]}"
    path = "#{params[:path]}"

    stats = Benchmark.realtime { @result = http_code("#{url}", "#{path}") }

      json = { :GlobalInformations => {
        :request_to              => "#{url}#{path}",
        :response_code           => "#{@result}",
        :response_time           => "#{stats}"
      }}
      JSON.pretty_generate(json)
  end

# Define methods
  def http_code(website, path)
    user_agent = 'Mozilla/5.0 (Heroku; availability check; get code)'
    @connect = Net::HTTP.new("#{website}", "80")
    resp, data = @connect.get("#{path}", {'User-Agent' => user_agent})
    resp.code
  end

end
