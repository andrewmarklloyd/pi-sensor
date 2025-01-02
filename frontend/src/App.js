import React from 'react';
import './App.css';
import HomePage from "./HomePage";
import SensorPage from "./SensorPage";
import ReportPage from "./ReportPage";
import "tabler-react/dist/Tabler.css";
import { BrowserRouter as Router, Route, Routes } from "react-router-dom";

function App() {
  return (
    <React.StrictMode>
      <Router>
        <Routes>
          <Route path="/" component={<HomePage/>} />
          <Route path="/sensor" component={<SensorPage/>} />
          <Route path="/report" component={<ReportPage/>} />
        </Routes>
      </Router>
    </React.StrictMode>
  );
}

export default App;
