GeoTTN
------

An application for The Things Network moving devices.

This is an all in one server, that will store historical data of your devices and display them on a map, also exposing an API to query.  

The idea is to have a self hosted solution IOT that works without sending your data to a third party.

The assumed encoding from your device is [Cayenne](https://developers.mydevices.com/cayenne/docs/lora/#lora-cayenne-low-power-payload)

## Technical Details

GeoTTN is a multi components app put together:

- A [Badger](https://github.com/dgraph-io/badger) storage database using [S2](https://s2geometry.io/) as a geographical indexing system
- A client of the Things Network gRPC API to receive uplink messages
- A gRPC API to query the Badger database
- A web frontend to display the devices on a map

It's shipped this way on purpose to run inside a one Docker instance, without resorting to complex admin task.  
It should be fine for thousands of devices until you want to break the components apart, contact me for a more robust clustered solution.

The code is modular so you can change it for your own purpose.

## Installation

You need to have some existing devices registered in the Things Network.  

Simply pass your `appID` & `appAccessKey` on the command line or via environment.

You can also use the docker image as follow:

```
docker run akhenakh/geottn:latest -e --tilesKey=pk.eyJxxxxxxxxxxxxxxxxxxxx  -e APPID=myappid -e APPACCESSKEY=xxxxxxxxxx -e DBPATH=/data/geo.db -v /mysafesotorage/volume:/data
```

For the map to show up register with MapBox for a [free token](https://account.mapbox.com/access-tokens/) and pass it as `tilesKey`.  
Note that you can use a [self hosted map solution](https://blog.nobugware.com/post/2019/self_hosted_world_maps/) with `selfHostedMap=true`.

## Plan

- Register devices with the web interface
- Vuejs web interface
- end to end TLS certs

## Help

This is a work in progress, help and ideas are welcome.