import { React, Component } from 'react';
import { Link } from "react-router-dom";
import { translateStatus, timeSince } from "./DataModel";
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';
import LockIcon from '@mui/icons-material/Lock';
import LockOpenIcon from '@mui/icons-material/LockOpen';
import Stack from '@mui/material/Stack';
import OfflineBoltIcon from '@mui/icons-material/OfflineBolt';

class Sensor extends Component {
  constructor(props) {
    super(props)
    var source = this.state.source
    var component = this
    this.props.socket.addEventListener("message", function(event) {
      var data = JSON.parse(event.data)
      if (data.channel === "sensor/status") {
        var d = JSON.parse(data.message)
        if (d.source === source) {
          var updated = translateStatus(d.status)
          component.setState({
            color: updated.color,
            source: d.source,
            icon: updated.icon,
            timestamp: d.timestamp,
            timesince: timeSince(d.timestamp)
          })
        }
      }
    })
  }

  state = {
    color: "",
    source: this.props.source,
    icon: "",
    timestamp: "",
    timesince: ""
  };

  componentDidMount() {
    var component = this
    setInterval(() => {
      component.setState({
        color: component.state.color !== "" ? component.state.color : component.props.color,
        source: component.state.source !== "" ? component.state.source : component.props.source,
        icon: component.state.icon !== "" ? component.state.icon : component.props.icon,
        timestamp: component.state.timestamp !== "" ? component.state.timestamp : component.props.timestamp,
        timesince: timeSince(component.state.timestamp !== "" ? component.state.timestamp : component.props.timestamp)
      })
    }, 60000)
  }

  render() {
    return (
      <Card sx={{ m: 0.5 }}>
        <CardContent>
          <Stack direction="row" spacing={3}>
            {getIcon(this)}
            <div>
              <Typography sx={{ fontSize: 20 }}>
                <Link
                to={{
                  pathname: "/sensor",
                  state: {
                    source: this.props.source,
                    timesince: this.props.timesince,
                    armed: this.props.armed,
                    timestamp: this.props.timestamp,
                    version: this.props.version
                  }
                }}>
                  {this.props.source}
                </Link>
              </Typography>
              <Typography>
                {this.state.timesince !== "" ? this.state.timesince : this.props.timesince}
              </Typography>
            </div>
          </Stack>
        </CardContent>
      </Card>
    );
  }
}

function getIcon(component) {
  let fill = component.state.color !== "" ? component.state.color : component.props.color
  let i = component.state.icon !== "" ? component.state.icon : component.props.icon
  if (i === "lock") {
    return <LockIcon style={{fill: fill}} />
  } else if (i === "unlock") {
    return <LockOpenIcon style={{fill: fill}} />
  } else {
    return <OfflineBoltIcon style={{fill: fill}} />
  }
}

export default Sensor;
