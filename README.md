# ical-transformer

A tool for renaming events and modifying metadata in iCal feeds. It hosts a web server that provides an iCal link for subscription.

## Installation

The most simple installation is using Docker. For that the Docker Compose file can be used. It uses the Dockerfile to build the image.

## Configuration

The tool can be configured using a toml file. The file should be called `transformer.toml`. An example is given below.

```TOML
# Transformer rules file

original = "https://ical-feed"
path = "/webcal/url-to-host-it"

# You can find the event id be looking in the original feed
[events."event-id@domain"]
summary = "New Title"
# Add dates where a recurring should NOT be scheduled. The format is YYYYMMDDTHHMMSS.
exdates_add = [ "20231024T080000", "20240116T080000", "20240123T080000", "20240130T080000", "20240206T080000", "20240213T080000" ]
```
