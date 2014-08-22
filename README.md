# request-map

<iframe width="420" height="315" src="//www.youtube.com/embed/fqXlebfqgaA" frameborder="0" allowfullscreen></iframe>

## How does it work?
This little tool:

1. reads logline on tcp/4223
2. resolves the location via geodb
3. sends the location of the IP to clients via websocket

## Use it
- compile
- download http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.mmdb.gz
- run
- visit http://localhost:2342
- tail some logfile via nc to localhost:4223 (`tail -f nginx.log | nc localhost 4223`)
