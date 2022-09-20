import { Outlet } from "react-router-dom";
import "./login_layout.scss";

import classNames from "classnames";

function LoginLayout() {
  return (
    <div className={classNames("login-layout")}>
      <Outlet />
    </div>
  );
}

export default LoginLayout;
