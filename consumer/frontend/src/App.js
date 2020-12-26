import { setupWebSocket, subscribeToChange } from './api';
import React, { Component } from 'react';
import './App.css';

class App extends Component {
  constructor(props) {
    super(props);
    setupWebSocket();
    subscribeToChange((data) => this.setState({
      data
    }));
  }

  state = {
    data: 'no data yet'
  };

  render() {
    return (
      <div className="App">
        <p className="App-intro">
        Door status: {this.state.data.state}
        </p>
      </div>
    );
  }
}

export default App;
