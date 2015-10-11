# incr

incr.it backend

## API

Submit metric value:

* `/:ns/:type/:metric/:value` - send any request (GET, POST etc) to
	submit new value. If sent by GET - a minimal transparent GIF is returned. For
	other requests only status is returned.

	* `:ns` - namespace, e.g. your account unique ID. GUID and MD5 hashsums are
		good examples, up to 64 chars.
	* `:type` - 'c' for counter, 'g' for gauge, 's' for set.
	* `:metric` - name of your metric. Any string up to 32 chars
	* `:value` - value submitted to the metric. Depends on metric type. Up to 64  
	  chars.

* `/:ns/:t/:m/:v/:ns2/:t2/:m2/:v2/:ns3/:t3/:m3/:v3/...` - bulk submit

Retrieve metric timeline:

* GET `/:ns` - returns all metrics in this namespace
* GET `/:ns/:t` - returns all metrics in this namespace by given type
* GET `/:ns/:t/:m` - returns single metric timeline

TCP, UDP:

* `/:ns/...` - submit
* `?:ns/...` - retrieve
