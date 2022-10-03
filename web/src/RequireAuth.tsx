import { useEffect } from "react";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import { Cookies } from "react-cookie";

const RequireAuth = () => {
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    const c = new Cookies();
    const cookies = c.get("torq_session");
    if (cookies === undefined) {
      navigate("/login", { replace: true, state: location });
    }
  });

  return <Outlet />;
}

export default RequireAuth
