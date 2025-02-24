import React from 'react';
import './App.css';
import ResponsiveAppBar from './components/AppBar';
import HomePage from "./HomePage";
import SensorPage from "./SensorPage";
import ReportPage from "./ReportPage";
import { BrowserRouter as Router, Route, Switch } from "react-router-dom";
import Footer from './components/Footer';

function App() {
  return (
    <React.StrictMode>
      <ResponsiveAppBar></ResponsiveAppBar>
      <Router>
        <Switch>
          <Route exact path="/" component={HomePage} />
          <Route exact path="/sensor" component={SensorPage} />
          <Route exact path="/report" component={ReportPage} />
        </Switch>
      </Router>
      <Footer></Footer>
    </React.StrictMode>
  );
}

export default App;
