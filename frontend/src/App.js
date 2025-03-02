import React from 'react';
import './App.css';
import ResponsiveAppBar from './components/AppBar';
import HomePage from "./HomePage";
import SensorPage from "./SensorPage";
import ReportPage from "./ReportPage";
import { BrowserRouter, Routes, Route } from "react-router";

import Footer from './components/Footer';

function App() {
  return (
    <React.StrictMode>
      <ResponsiveAppBar></ResponsiveAppBar>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<HomePage/>} />
          <Route path="/sensor" element={<SensorPage/>} />
          <Route path="/report" element={<ReportPage/>} />
        </Routes>
      </BrowserRouter>
      <Footer></Footer>
    </React.StrictMode>
  );
}

export default App;
