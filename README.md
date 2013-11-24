nflx-proxy
==========
A DNS proxy for accessing Netflix in different regions.

Example
-------
* Setup a server with an IP address in the region you want to access. For
  example a EC2 instance in a US region.

* Start nflix-proxy and map two Netflix zones to your Amazon EC2 public IP
  address:

```
$ nflix-proxy movies.netflix.com:1.3.3.7 cbp-us.nccp.netflix.com:1.3.3.7
2013/11/24 19:08:41 Answering movies.netflix.com. with 1.3.3.7
2013/11/24 19:08:41 Answering cbp-us.nccp.netflix.com. with 1.3.3.7
 ```

* Set the IP of your server as your computers (or networks) DNS
