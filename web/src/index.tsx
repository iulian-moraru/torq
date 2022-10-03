import React from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "react-redux";
import { BrowserRouter } from "react-router-dom";

import App from "./App";
import reportWebVitals from "./reportWebVitals";
import { store } from "./store/store";

import "./styles/main.scss";

const resolveBasename = () => {
  const path = window.location.pathname.split("/");
  const topLevelPath = path.length > 1 && path[1];

  return topLevelPath === "torq" ? topLevelPath : "";
};

const basename = resolveBasename();

const appContainer = document.getElementById("root");
const root = createRoot(appContainer!);

root.render(
  <React.StrictMode>
    <BrowserRouter basename={basename}>
      <Provider store={store}>
        <App />
      </Provider>
    </BrowserRouter>
  </React.StrictMode>
);

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
