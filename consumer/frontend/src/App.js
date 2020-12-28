import React, { Component } from 'react';
import './App.css';
import HomePage from "./HomePage";
import "tabler-react/dist/Tabler.css";

class App extends Component {
  render() {
    return (
      <div className="App">
        <p className="App-intro">
        <HomePage/>
        </p>
      </div>
    );
  }
}

export default App;
