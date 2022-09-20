import React from "react";
import { ReactComponent as TorqLogo } from "../../icons/torq-logo.svg";
import { LockOpen20Regular as UnlockIcon } from "@fluentui/react-icons";
import "./login_page.scss";
import { useLocation, useNavigate } from "react-router-dom";
import { useLoginMutation } from "apiSlice";

function LoginPage() {
  const [login] = useLoginMutation();

  const navigate = useNavigate();
  const location = useLocation();
  interface LocationState {
    from: {
      pathname: string;
    };
  }

  let from = (location.state as LocationState)?.from?.pathname || "/";
  // Don't redirect back to logout.
  if (from === "/logout" || from === "/login" || from === "logout" || from === "login" || from === "" || from === "/") {
    from = "/";
  }

  const submit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const formData = new FormData(e.currentTarget);
    formData.append("username", "admin");
    await login(formData);
    navigate(from, { replace: true });
  };

  // TODO: unify the styling here once standardised button styles are done.
  return (
    <div className="login-page-wrapper">
      <div className="login-form-wrapper">
        <div className="logo">
          <TorqLogo />
        </div>
        <form className="login-form" onSubmit={submit}>
          <input type="password" name={"password"} className={"password-field"} placeholder={"Password..."} />
          <button type="submit" className={"submit-button"}>
            <UnlockIcon />
            Login
          </button>
        </form>
      </div>
    </div>
  );
}

export default LoginPage;
