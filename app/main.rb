require 'rubygems'
require 'sinatra/base'
require 'benchmark'
require 'json'
require 'net/http'
require 'net/https'
require 'uri'

class NetChecks < Sinatra::Base

# If try to access to root, do redirect => skillstar.com
# HTTP Response : 302
  get '/' do
    redirect 'https://www.dotinfra.fr'
  end

# Getting JSON serialized return of checks
  get '/checks' do
    content_type :json

    url = "#{params[:url]}"
    proxy = "#{params[:proxy]}"

    stats = Benchmark.realtime { @result = http_get(URI(url), URI(proxy)) }

      json = { :HTTP_Check => {
        :request_to              => "#{url}",
        :response_code           => "#{@result.code}",
        :response_time           => "#{stats}",
        :cachecontrol            => "#{@result.header.get_fields('Cache-Control').first.to_s}"
      }
      }
      JSON.pretty_generate(json)
  end

# Define methods
  def http_get(uri, proxy)
    user_agent = 'Mozilla/5.0 (dotINFRA; remote check)'
    connect = Net::HTTP.new(uri.host, uri.port, proxy.host, proxy.port )
    req = Net::HTTP::Get.new(uri.request_uri, {'User-Agent' => user_agent})
    req.basic_auth uri.user, uri.password
    resp, data = connect.request(req)
  end

end
