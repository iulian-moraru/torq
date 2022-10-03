import React from "react";
import styles from "./nav.module.scss";
import classNames from "classnames";
import { NavLink, useMatch, useResolvedPath } from "react-router-dom";

function MenuItem(props: { text: string; icon?: any; routeTo: string }) {
  const resolvedPath = useResolvedPath(props.routeTo);
  const hasMatch = useMatch({ path: resolvedPath.pathname, end: true });

  const resolvedClassNames = classNames(styles.title, { [styles.selected] : hasMatch });

  return (
    <div className={classNames(styles.item)}>
      <div className={classNames(styles.contentWrapper)}>
        <NavLink to={props.routeTo} className={resolvedClassNames}>
          <div className={styles.icon}>{props.icon}</div>
          <div className={classNames(styles.text)}>{props.text}</div>
        </NavLink>
      </div>
    </div>
  );
}

export default React.memo(MenuItem);
