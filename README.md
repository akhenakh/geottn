GeoTTN
------

An application for [The Things Network](https://www.thethingsnetwork.org/) moving devices.

This is an all in one server, that will store historical data of your devices and display them on a map, also exposing an API to query.  

The idea is to have a self hosted IOT solution that works without sending your data to a third party.

The assumed encoding from your device is [Cayenne](https://developers.mydevices.com/cayenne/docs/lora/#lora-cayenne-low-power-payload)


![Current web interface](/img/interface.jpg?raw=true "Inteface")

## Technical Details

GeoTTN is a multi components app put together:

- A [Badger](https://github.com/dgraph-io/badger) storage database using [S2](https://s2geometry.io/) as a geographical indexing system
- A client of the Things Network gRPC API to receive uplink messages
- A gRPC API to query the Badger database
- A web frontend to display the devices on a map

It's shipped in one app on purpose to run inside a one Docker instance, without resorting to complex admin task.  
It should be fine for thousands of devices until you want to break the components apart, contact me for a more robust clustered solution.

The code is modular so you can change it for your own purpose.

## Installation

You need to have some existing devices registered in the Things Network.  

Simply pass your `appID` & `appAccessKey` on the command line or via environment.

You can also use the docker image as follow:

```
docker run -it akhenakh/geottn:latest -e TILESKEY=pk.eyJxxxxxxxxxxxxxxxxxxxx  -e APPID=myappid -e APPACCESSKEY=xxxxxxxxxx -e DBPATH=/data/geo.db -v /mysafesotorage/volume:/data
```

For the map to show up register with MapBox for a [free token](https://account.mapbox.com/access-tokens/) and pass it as `tilesKey`.  
Note that you can use a [self hosted map solution](https://blog.nobugware.com/post/2019/self_hosted_world_maps/) with `selfHostedMap=true`.



## Build

You'll need the `packr2` command:
```
go get -u github.com/gobuffalo/packr/v2/packr2
```

```
make geottnd
make geottnd-image
```

## API
A gRPC API is exposed 

```proto
service GeoTTN {
  rpc Store (DataPoint) returns (google.protobuf.Empty) {}
  rpc RadiusSearch(RadiusSearchRequest) returns (DataPoints) {}
  rpc RectSearch(RectSearchRequest) returns (DataPoints) {}
  rpc Get(GetRequest) returns (DataPoint) {}
  rpc GetAll(GetRequest) returns (DataPoints) {}
  rpc Keys(google.protobuf.Empty) returns (KeyList) {}
}
```

There is a demo cli in `cmd/geottncli`

```
 ./cmd/geottncli/geottncli -radius=1000
2019/11/22 14:28:03 query ok 2
2019/11/22 14:28:03 map[device_id:ttgo00 gps_1:[48.4 2.45 0] time:seconds:1574438932 nanos:728890266 ]
2019/11/22 14:28:03 map[device_id:ttgosens00 gps_1:[48.8821 2.28 0] time:seconds:1574434228 nanos:790831992 ]

./cmd/geottncli/geottncli -key ttgosens00  
2019/11/22 14:28:19 map[device_id:ttgosens00 gps_1:[48.8821 2.28 0] time:seconds:1574434228 nanos:790831992 ]

```

A very simple web API (used for the web interface):
```go
r.HandleFunc("/api/devices", s.DevicesQuery)
r.HandleFunc("/api/data/{key}", s.DataQuery)
r.HandleFunc("/api/rect/{urlat}/{urlng}/{bllat}/{bllng}", s.RectQuery)
```

## Stats

Some stats are available on the metrics ports `httpMetricsPort` eg `http://localhost:8888/metrics`

## Plan

- Register devices with the web interface
- Vuejs web interface
- end to end TLS certs
- support no GPS data

## Help

This is a work in progress, help and ideas are welcome.