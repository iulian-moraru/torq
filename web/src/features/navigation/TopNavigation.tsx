import { useAppDispatch } from "../../store/hooks";
import { toggleNav } from "./navSlice";
import classNames from "classnames";
import { ReactComponent as TorqLogo } from "../../icons/torq-logo.svg";
import { Navigation20Regular as CollapseIcon } from "@fluentui/react-icons";
import styles from "./nav.module.scss";

function TopNavigation() {
  const dispatch = useAppDispatch();
  return (
    <div className={classNames(styles.topNavigation)}>
      <div className={classNames(styles.topLogo)}>
        <TorqLogo />
      </div>

      {/*<div className={classNames(styles.eventsButton)}>*/}
      {/*  <EventsIcon />*/}
      {/*</div>*/}

      <div className={styles.topCollapseButton} onClick={() => dispatch(toggleNav())}>
        <CollapseIcon />
      </div>
    </div>
  );
}

export default TopNavigation;
