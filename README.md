# spectrum-capture
spectrum-capture is part of the micboard-spectrum prototype. Additional information can be found in the [spectrum-frontend](https://github.com/karlcswanson/spectrum-frontend) project. 

spectrum-capture is a wrapper for soapy_power and other utilities that provide rtl_power formatted data via stdio.  spectrum-capture streams incremental and complete scan data to a mqtt broker. 

## Configuration
A single json object is passed in via SPECTRUM_CFG in `.env`. This json snippet contains the following parameters:

* Name
* Description
* Command
* Servers
  * broker
  * client_id

```
SPECTRUM_CFG={ "name":"Lititz, PA", "description":"Pluto", "command" : "soapy_power -f 470M:608M -B 25k -c -k 50 -g 50 -r 2M", "servers" : [{ "broker" : "mqtt://localhost:1883", "client_id" : "06865db2-f4eb-4e59-9809-ef2b9b5af745"}]}
```

## Remote Development
For convenience, a remote deployment script is included in this repo. This script compiles `spectrum-capture` to an arm64 binary, copies the file to a remote device, and starts `spectrum-capture` in a remote [soapySDR-container](https://github.com/karlcswanson/soapySDR-container).