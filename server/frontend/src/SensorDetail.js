import React, { Component } from 'react';
import { Button, Card } from "tabler-react";

class SensorDetail extends Component {

    render() {
        return (
        <Card>
            <Card.Header>
                <Card.Title>Card title</Card.Title>
            </Card.Header>
            <Card.Body>
                Body of the card here
            </Card.Body>
            <Card.Footer>This is standard card footer</Card.Footer>
        </Card>
        )
    }
}


export default SensorDetail;
