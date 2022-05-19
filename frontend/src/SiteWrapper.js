import * as React from "react";

import {
  Site,
  Nav,
} from "tabler-react";

// https://github.com/tabler/tabler-react/blob/6981fe1f1710011a57201b9547d280d86daa3f41/example/src/data/icons/fe.json
const navBarItems = [
  {
    value: "Home",
    to: "/",
    icon: "home",
    useExact: true,
  },
  {
    value: "Report",
    to: "/report",
    icon: "bar-chart",
    useExact: true,
  },
  {
    value: "Logout",
    to: "/logout",
    icon: "log-out",
    useExact: true,
  }
];

class SiteWrapper extends React.Component {
  render() {
    return (
      <Site.Wrapper
        headerProps={{
          href: "/",
          alt: "Pi Sensor",
          navItems: (
            <Nav.Item type="div" className="d-none d-md-flex">
            </Nav.Item>
          ),
        }}
        navProps={{ itemsObjects: navBarItems }}
        footerProps={{
          links: [
            <a href="https://github.com/andrewmarklloyd/pi-sensor">Source Code</a>,
            <a href="https://github.com/tabler/tabler-react">Built with Tabler React</a>
          ],
          note:
            <a href={"https://github.com/andrewmarklloyd/pi-sensor/commit/"+process.env.REACT_APP_VERSION}>App Version {process.env.REACT_APP_VERSION}</a>
        }}
      >
        {this.props.children}
      </Site.Wrapper>
    );
  }
}

export default SiteWrapper;
