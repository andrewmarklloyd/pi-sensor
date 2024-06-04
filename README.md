# pi-sensor

![example workflow](https://github.com/andrewmarklloyd/pi-sensor/actions/workflows/main.yml/badge.svg)


Distributed magnetic sensor status dashboard and alerting system. Allows an arbitrary number of Raspberry Pi Zero's to send the status of a magnetic sensor to a messaging bus where a server component hosted on Digital Ocean aggregates the statuses along with other information in a dashboard. Alerting is also enabled to send notifications through Home Assistant on sensor status changes.

### Server

Golang server using Mosquitto messaging and Redis for data storage.

### Agent

Raspberry Pi Zero using a magnetic sensor to detect open and closed doors, windows
