import React from "react";
import styles from "./note.module.scss";
import { Note24Regular as DefaultNoteIcon } from "@fluentui/react-icons";
import classNames from "classnames";

type NoteProps = {
  title: string;
  icon?: React.ReactNode;
  className?: string;
  children: React.ReactNode;
};

function Note(props: NoteProps) {
  return (
    <div className={classNames(styles.noteWrapper, props.className)}>
      <div className={styles.noteTitleWrapper}>
        <div className={styles.noteTitleIcon}>{props.icon || <DefaultNoteIcon />}</div>
        <div className={styles.noteTitle}>{props.title}</div>
      </div>
      <div className={styles.noteContent}>{props.children}</div>
    </div>
  );
}

export default Note;
