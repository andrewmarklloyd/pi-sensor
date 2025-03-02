import { React, useState } from "react";
import { Link, useLocation } from "react-router";
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Button from '@mui/material/Button';
import { trimVersion, unixToDate } from "./DataModel";

const SensorPage = () => {
  let location = useLocation()
  const [state, setState] = useState(location.state)

  function toggleArm(source) {
    if (window.confirm('Are you sure you wish to toggle arm/disarm?')) {
      fetch("/api/sensor/arming", {
        method: 'POST',
        credentials: 'same-origin',
        headers: {
          'Content-Type': 'application/json'
        },
        referrerPolicy: 'no-referrer',
        body: JSON.stringify({source: source})
      })
      .then(r => r.json())
      .then(res => {
        state.armed = res.armed
        setState(state);
      })
    }
  }

  function handleChange(a) {
    state.openTimeout = parseInt(a.target.value)
    setState(state)
  }

  function submitOpenTimeout(event) {
    event.preventDefault();
    fetch("/api/sensor/openTimeout", {
      method: 'POST',
      credentials: 'same-origin',
      headers: {
        'Content-Type': 'application/json'
      },
      referrerPolicy: 'no-referrer',
      body: JSON.stringify({source: state.source, openTimeout: state.openTimeout})
    })
    .then(r => r.json())
    .then(res => {
      if (res.status === "success") {
        alert("Successfully updated open timeout")
      } else {
        alert("Error updating open timeout: " + res.error)
      }
    })
  }

  fetch("/api/sensor/getOpenTimeout?source="+state.source, {
    method: 'GET',
    credentials: 'same-origin',
    headers: {
      'Content-Type': 'application/json'
    },
    referrerPolicy: 'no-referrer'
  })
  .then(r => r.json())
  .then(res => {
    if (res.status === "success") {
      state.openTimeout = res.openTimeout
      setState(state)
    } else {
      console.log("error getting openTimeout: ", res.error)
    }
  })

  return (
    <div>
    <Card sx={{ m: 0.5 }}>
      <CardContent>
        <h2>Sensor: {state.source}</h2>
        <p>Last activity: {unixToDate(state.timestamp)}</p>
            <p>Alerting: {state.armed === "true" ? "Armed" : "Disarmed"}</p>
            <p>Version: {trimVersion(state.version)}</p>
            <div>
              <button onClick={() => toggleArm(state.source)}>
                Arm/Disarm
              </button>
            </div>
            <div>
              <form onSubmit={submitOpenTimeout.bind(this)}>
                <label>
                  Open Timeout:
                  <input type="number" min="1" max="60" value={state.openTimeout} onChange={handleChange.bind(this)} />
                </label>
                <input type="submit" value="Submit" />
              </form>
            </div>
      </CardContent>
    </Card>
    <Link to={{pathname: "/"}}><Button variant="outlined" >Back</Button></Link>
  </div>
  )
}

export default SensorPage;
