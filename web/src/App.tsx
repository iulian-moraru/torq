import React from "react";

import styles from "./app.module.scss";
import ToastContext from "./features/toast/context";
import Toasts, { addToastHandle } from "./features/toast/Toasts";
import Router from "./Router";

function App() {
  const toastRef = React.useRef<addToastHandle>();

  return (
    <ToastContext.Provider value={toastRef}>
      <div className={styles.app}>
        <Toasts ref={toastRef} />
        <Router />
      </div>
    </ToastContext.Provider>
  );
}

export default App;
