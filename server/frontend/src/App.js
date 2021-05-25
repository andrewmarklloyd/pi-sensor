import React from 'react';
import './App.css';
import HomePage from "./HomePage";
import "tabler-react/dist/Tabler.css";
import { BrowserRouter as Router, Route, Switch } from "react-router-dom";

function App() {
  return (
    <React.StrictMode>
      <Router>
        <Switch>
          <Route exact path="/" component={HomePage} />
          <Route exact path="/sensor" component={HomePage} />
        </Switch>
      </Router>
    </React.StrictMode>
  );
}

export default App;