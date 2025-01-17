import { useEffect } from "react";
import { Cookies } from "react-cookie";
import { RouteObject, useRoutes } from "react-router";
import { useLocation, useNavigate } from "react-router-dom";

import { useLogoutMutation } from "apiSlice";
import RequireAuth from "RequireAuth";

import DefaultLayout from "layout/DefaultLayout";
import LoginLayout from "layout/LoginLayout";

import LoginPage from "features/auth/LoginPage";
import ChannelPage from "features/channel/ChannelPage";
import ChannelsPage from "features/channels/ChannelsPage";
import DashboardPage from "features/channel/DashboardPage";
import ForwardsPage from "features/forwards/ForwardsPage";
import NoMatch from "features/no_match/NoMatch";
import SettingsPage from "features/settings/SettingsPage";
import AllTxPage from "features/transact/AllTxPage";
import InvoicesPage from "features/transact/Invoices/InvoicesPage";
import OnChainPage from "features/transact/OnChain/OnChainPage";
import NewPaymentModal from "features/transact/Payments/newPayment/NewPaymentModal";
import NewAddressModal from "features/transact/OnChain/newAddress/NewAddressModal";
import UpdateChannelModal from "features/channels/updateChannel/UpdateChannelModal";
import PaymentsPage from "features/transact/Payments/PaymentsPage";

import * as routes from "constants/routes";
import NewInvoiceModal from "./features/transact/Invoices/newInvoice/NewInvoiceModal";

function Logout() {
  const [logout] = useLogoutMutation();
  const navigate = useNavigate();

  useEffect(() => {
    const c = new Cookies();
    c.remove("torq_session");
    logout();
    navigate("/login", { replace: true });
  });

  return <div />;
}

const publicRoutes: RouteObject = {
  element: <LoginLayout />,
  children: [
    { path: routes.LOGIN, element: <LoginPage /> },
    { path: routes.LOGOUT, element: <Logout /> },
  ],
};

const modalRoutes: RouteObject = {
  children: [
    { path: routes.NEW_INVOICE, element: <NewInvoiceModal /> },
    { path: routes.NEW_PAYMENT, element: <NewPaymentModal /> },
    { path: routes.NEW_ADDRESS, element: <NewAddressModal /> },
    { path: routes.UPDATE_CHANNEL, element: <UpdateChannelModal /> },
  ],
};

const authenticatedRoutes: RouteObject = {
  element: <DefaultLayout />,
  children: [
    {
      element: <RequireAuth />,
      children: [
        {
          path: routes.ROOT,
          element: <DashboardPage />,
          children: modalRoutes.children,
        },
        {
          path: routes.ANALYSE,
          children: [
            { path: routes.CHANNELS, element: <ChannelsPage /> },
            { path: routes.FORWARDS, element: <ForwardsPage /> },
            { path: routes.FORWARDS_CUSTOM_VIEW, element: <ForwardsPage /> },
            { path: routes.INSPECT_CHANNEL, element: <ChannelPage /> },
          ],
        },
        {
          path: routes.TRANSACTIONS,
          children: [
            { path: routes.PAYMENTS, element: <PaymentsPage /> },
            { path: routes.INVOICES, element: <InvoicesPage /> },
            { path: routes.ONCHAIN, element: <OnChainPage /> },
            { path: routes.ALL, element: <AllTxPage /> },
          ],
        },
        { path: routes.SETTINGS, element: <SettingsPage /> },
        { path: "*", element: <NoMatch /> },
      ],
    },
  ],
};

const Router = () => {
  const location = useLocation();
  const background = location.state && location.state.background;
  const currentLocation = background || location;

  const routes = [publicRoutes, authenticatedRoutes];

  const router = useRoutes(routes, currentLocation);
  const modalRouter = useRoutes([modalRoutes]);

  return (
    <>
      {router}
      {background && modalRouter}
    </>
  );
};

export default Router;
