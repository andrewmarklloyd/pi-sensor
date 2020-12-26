import { setupWebSocket } from './api';
import React, { Component } from 'react';
import './App.css';

class App extends Component {
  constructor(props) {
    super(props);
    setupWebSocket();
  }

  state = {
    timestamp: 'no timestamp yet'
  };

  render() {
    return (
      <div className="App">
        <p className="App-intro">
        This is the timer value: {this.state.timestamp}
        </p>
      </div>
    );
  }
}

export default App;
