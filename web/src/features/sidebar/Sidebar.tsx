import React from "react";
import { DismissCircle24Regular as CloseIcon, Options24Regular as SideBarIcon } from "@fluentui/react-icons";

import styles from "./sidebar.module.scss";

type SidebarProps = {
  title: string;
  children?: React.ReactNode;
  closeSidebarHandler: (event: React.MouseEvent<HTMLDivElement, MouseEvent>) => void;
};

function Sidebar(props: SidebarProps) {
  return (
    <div className={styles.sidebarWrapper}>
      <div className={styles.titleContainer}>
        <div className={styles.icon}>
          <SideBarIcon />
        </div>
        <div className={styles.title}>{props.title}</div>
        <div className={styles.close} onClick={props.closeSidebarHandler}>
          <CloseIcon />
        </div>
      </div>
      <div className={styles.sidebarContent}>{props.children}</div>
    </div>
  );
}

export default Sidebar;
