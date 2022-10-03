import React from "react";
import styles from "./popoutPageTemplate.module.scss";
import { DismissCircle24Regular as DismissIcon } from "@fluentui/react-icons";
import classNames from "classnames";

type PopoutPageTemplateProps = {
  children: React.ReactNode;
  show: boolean;
  title?: string;
  icon?: React.ReactNode;
  onClose: () => void;
};

const PopoutPageTemplate = (props: PopoutPageTemplateProps) => {
  const handleClose = () => {
    props.onClose();
  };
  const ignore = (e: React.MouseEvent<HTMLDivElement>) => {
    e.stopPropagation();
    e.preventDefault();
  };
  return (
    <div className={classNames(styles.modal, { [styles.show]: props.show })}>
      <div className={styles.modalBackdrop} onClick={handleClose} />

      <div className={styles.popoutWrapper} onClick={ignore}>
        <div className={styles.header}>
          {props.icon && <span className={styles.icon}>{props.icon}</span>}
          <span className={styles.title}>{props.title}</span>
          <span className={styles.close} onClick={handleClose}>
            <DismissIcon />
          </span>
        </div>
        <div className={styles.contentWrapper}>{props.children}</div>
      </div>
    </div>
  );
};

export default PopoutPageTemplate;
