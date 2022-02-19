import urllib
import urllib2
 
url = 'http://127.0.0.1:4444/dojo.txt'
data = '{ "jsonrpc" : "2.0", "id" : 12345, "method" : "align_isdone", "params" : { } }'
 
req = urllib2.Request(url, data)
response = urllib2.urlopen(req)
r = response.read()
print r

