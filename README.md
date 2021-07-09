# pi-sensor

Distributed magnetic sensor status dashboard and alerting system. Allows an arbitrary number of Raspberry Pi Zero's to send the status of a magnetic sensor to a messaging bus where a server component hosted on Heroku aggregates the statuses along with other information in a dashboard. Alerting is also enabled to send text messages on sensor status changes.

### Server

Golang server using MQTT messaging and Redis for data storage.

### Client

Raspberry Pi Zero using a magnetic sensor to detect open and closed doors, windows

Install client on Raspberry Pi Zero.

```
bash <(curl -s -H 'Cache-Control: no-cache' https://raw.githubusercontent.com/andrewmarklloyd/pi-sensor/master/install/install-client.sh)
```

### TODO

- Fill in full readme
- Convert frontend to dark mode
- Add testing
- Arm/Disarm
    - Need more dynamic state of SensorPage
