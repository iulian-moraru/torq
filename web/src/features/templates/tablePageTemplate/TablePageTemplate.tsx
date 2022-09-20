import React from "react";
import styles from "./table-page-template.module.scss";

import classNames from "classnames";
import { FluentIconsProps } from "@fluentui/react-icons";
import PageTitle from "features/templates/PageTitle";
import Button, { buttonColor } from "../../buttons/Button";

type TablePageTemplateProps = {
  title: string;
  titleContent?: React.ReactNode;
  sidebarExpanded?: boolean;
  sidebar?: React.ReactNode;
  pagination?: React.ReactNode;
  pageTotals?: React.ReactNode;
  tableControls?: React.ReactNode;
  breadcrumbs?: Array<any>;
  children?: React.ReactNode;
};

export default function TablePageTemplate(props: TablePageTemplateProps) {
  return (
    <div className={styles.contentWrapper}>
      <PageTitle breadcrumbs={props.breadcrumbs} title={props.title}>
        {props.titleContent}
      </PageTitle>

      <div className={classNames(styles.totalWrapper)}>{props.pageTotals}</div>

      {props.tableControls}

      <div className={styles.tableWrapper}>
        <div className={styles.tableContainer}>
          <div className={styles.tableExpander}>{props.children}</div>
        </div>
      </div>

      <div className={classNames(styles.paginationWrapper)}>{props.pagination}</div>

      <div className={classNames(styles.pageSidebarWrapper, { [styles.sidebarExpanded]: props.sidebarExpanded })}>
        {props.sidebar}
      </div>
    </div>
  );
}

type TableControlsButtonProps = {
  icon: React.FC<FluentIconsProps>;
  onClickHandler?: () => void;
  active?: boolean;
};

export function TableControlsButton(props: TableControlsButtonProps) {
  return (
    <div className={classNames(styles.tableControlsButtonWrapper, { [styles.active]: props.active })}>
      <Button
        text={""}
        className={styles.tableControlsButtonIcon}
        onClick={props.onClickHandler}
        buttonColor={buttonColor.subtle}
        icon={<props.icon />}
      />
      {/*<div className={styles.title}>{props.title}</div>*/}
    </div>
  );
}

export function TableControlSection(props: { children?: React.ReactNode }) {
  return <div className={classNames(styles.tableControlsSection)}>{props.children}</div>;
}

export function TableControlsButtonGroup(props: { children?: React.ReactNode }) {
  return <div className={classNames(styles.tableControlsButtonGroup)}>{props.children}</div>;
}

export function TableControlsTabsGroup(props: { children?: React.ReactNode }) {
  return <div className={classNames(styles.tableControlsTabsGroup)}>{props.children}</div>;
}
