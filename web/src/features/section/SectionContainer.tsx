import styles from "./sectionContainer.module.scss";
import {
  ChevronDown20Regular as CollapsedIcon,
  FluentIconsProps,
  LineHorizontal120Regular as ExpandedIcon,
} from "@fluentui/react-icons";
import classNames from "classnames";
import React from "react";

type SectionContainerProps = {
  title: string;
  icon: React.FC<FluentIconsProps>;
  children: React.ReactNode;
  expanded?: boolean;
  disabled?: boolean;
  handleToggle?: (event: React.MouseEvent<HTMLDivElement, MouseEvent>) => void;
};

export function SectionContainer(props: SectionContainerProps) {
  return (
    <div className={classNames(styles.sectionContainer, { [styles.disabled]: props.disabled })}>
      <div className={styles.sectionTitleContainer} onClick={props.handleToggle}>
        <div className={styles.sidebarIcon}>
          <props.icon />
        </div>
        <div className={styles.sidebarTitle}>{props.title}</div>
        <div className={styles.sidebarIcon}>{props.expanded ? <ExpandedIcon /> : <CollapsedIcon />}</div>
      </div>
      <div className={classNames(styles.sidebarSectionContent, { [styles.expanded]: props.expanded })}>
        {props.children}
      </div>
    </div>
  );
}
