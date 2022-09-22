import React from "react";
import styles from "./Modal.module.scss";
import { DismissCircle24Regular as DismissIcon } from "@fluentui/react-icons";
import classNames from "classnames";

interface ModalProps {
  children: React.ReactNode;
  show: boolean;
  title?: string;
  icon?: React.ReactNode;
  onClose: () => void;
}

const Modal = (props: ModalProps) => {
  const handleClose = () => {
    props.onClose();
  };
  const ignore = (e: React.MouseEvent<HTMLDivElement>) => {
    e.stopPropagation();
  };
  return (
    <div className={classNames(styles.modal, { [styles.show]: props.show })} onClick={handleClose}>
      <div className={styles.content} onClick={ignore}>
        <div className={styles.header}>
          {props.icon && <span className={styles.icon}>{props.icon}</span>}
          <span className={styles.title}>{props.title}</span>
          <span className={styles.close} onClick={handleClose}>
            <DismissIcon />
          </span>
        </div>
        {props.children}
      </div>
    </div>
  );
};

export default Modal;
