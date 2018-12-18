# rail

## Register for National Rail Data Portal

* https://datafeeds.nationalrail.co.uk/darwin/index.html#/registration
  * User type: *Personal*
  * Planned usage: *Personal Project*
  * Usage details: *Other: Personal Use*
* Once you've registered, navigate to My Feeds and tweak:
  * Darwin Data Types:
    * *Train Actuals / Predictions*
  * Dawnin TIPLOCs:
    * *Select my TIPLOCs*
    * Add the station(s) of interest from www.railwaycodes.org.uk

NOTE: Make sure you have your RTF Queue Name hands for exporting below.

## Give it a whirl

```
export GO111MODULE=yes
export RAIL_QUEUE_NAME="$RTF_QUEUE_NAME"

go run main.go
```
