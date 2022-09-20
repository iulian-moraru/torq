import { useAppDispatch } from "../../store/hooks";
import { toggleNav } from "./navSlice";
import classNames from "classnames";
import MenuItem from "./MenuItem";
import NavCategory from "./NavCategory";
import { ReactComponent as TorqLogo } from "../../icons/torq-logo.svg";
import {
  Navigation20Regular as CollapseIcon,
  AppsListDetail20Regular as ForwardsIcon,
  Autosum20Regular as SummaryIcon,
  MoneyHand20Regular as TransactionIcon,
  LockClosed20Regular as LogoutIcon,
  Settings20Regular as SettingsIcon,
} from "@fluentui/react-icons";
import styles from "./nav.module.scss";

function Navigation() {
  const dispatch = useAppDispatch();

  return (
    <div className={classNames(styles.navigation)}>
      <div className={styles.logoWrapper}>
        <div className={classNames(styles.logo)}>
          <TorqLogo />
        </div>

        {/*<div className={classNames(styles.eventsButton)}>*/}
        {/*  <EventsIcon />*/}
        {/*</div>*/}

        <div className={styles.collapseButton} id={"collapse-navigation"} onClick={() => dispatch(toggleNav())}>
          <CollapseIcon />
        </div>
      </div>

      <div className={styles.mainNavWrapper}>
        {/*<MenuItem text={"Dashboard"} icon={<DashboardIcon />} routeTo={"/sadfa"} />*/}

        <NavCategory text={"Analyse"} collapsed={false}>
          <MenuItem text={"Summary"} icon={<SummaryIcon />} routeTo={"/"} />
          <MenuItem text={"Forwards"} icon={<ForwardsIcon />} routeTo={"/analyse/forwards"} />
          {/*<MenuItem text={"Inspect"} icon={<InspectIcon />} routeTo={"/inspect"} />*/}
        </NavCategory>

        {/*<NavCategory text={"Manage"} collapsed={false}>*/}
        {/*  <>*/}
        {/*    <MenuItem text={"Nodes"} icon={<NodesIcon />} routeTo={"/nodes"} />*/}
        {/*    <MenuItem text={"Channels"} icon={<ChannelsIcon />} routeTo={"/channelss"} />*/}
        {/*  </>*/}
        {/*</NavCategory>*/}

        <NavCategory text={"Transact"} collapsed={false}>
          <MenuItem text={"Transactions"} icon={<TransactionIcon />} routeTo={"/transactions/payments"} />
        </NavCategory>
      </div>

      <div className={classNames(styles.bottomWrapper)}>
        <MenuItem text={"Settings"} icon={<SettingsIcon />} routeTo={"/settings"} />
        <MenuItem text={"Logout"} icon={<LogoutIcon />} routeTo={"/logout"} />
      </div>
    </div>
  );
}

export default Navigation;
