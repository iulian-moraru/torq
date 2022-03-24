import { Routes, Route, Link } from "react-router-dom";
import { useSelector } from "react-redux";

import Navigation from "./components/navigation/Navigation";
import TablePage from "./pages/TablePage";
import './App.scss';
import classNames from "classnames";

function App() {

  const navHidden: number = useSelector((state: { navHidden: number }) => { return state.navHidden });
  return (
    <div className="App torq">
      <div className={classNames("main-content-wrapper", { 'nav-hidden': navHidden })}>
        <div className="navigation-wrapper">
          <Navigation />
        </div>
        <div className="page-wrapper">
          <Routes>
            <Route path="/" element={<TablePage />} />
          </Routes>
        </div>
      </div>
    </div>
  );
}

export default App;
