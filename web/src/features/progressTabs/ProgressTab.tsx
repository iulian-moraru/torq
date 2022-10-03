import styles from "./progressTabs.module.scss";
import React from "react";

type ProgressTabContainerProps = {
  // modalCloseHandler: Function;
  children: React.ReactNode;
};

export function ProgressTabContainer(props: ProgressTabContainerProps) {
  return <div className={styles.progressTabWrapper}>{props.children}</div>;
}

type ProgressTabsProps = {
  // modalCloseHandler: Function;
  showTabIndex: number;
  children: Array<React.ReactNode> | React.ReactNode;
};
//
const cssGridTemplate = (index: number) => {
  return { transform: `translateX(calc(-${index * 100}% - ${index * 48}px))` };
};

function ProgressTabs(props: ProgressTabsProps) {
  return (
    <div className={styles.progressTabsWrapper}>
      <div className={styles.progressTabsContainer} style={cssGridTemplate(props.showTabIndex)}>
        {!Array.isArray(props.children)
          ? props.children
          : props.children.map((step, index) => {
              return step;
            })}
      </div>
    </div>
  );
}

export default ProgressTabs;
