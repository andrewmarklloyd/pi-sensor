import * as React from "react";

import {
  Site,
  Nav,
} from "tabler-react";

const navBarItems = [
  {
    value: "Home",
    to: "/",
    icon: "home",
    // LinkComponent: withRouter(NavLink),
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
            <a href="https://github.com/tabler/tabler-react">Tabler React</a>
          ],
          note:
            "Site designed using Tabler React",
        }}
      >
        {this.props.children}
      </Site.Wrapper>
    );
  }
}

export default SiteWrapper;
