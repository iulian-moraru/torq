import styles from "./switch.module.scss";
import classNames from "classnames";
import React from "react";

type SwitchProps = {
  label: string;
  checked?: boolean;
  onChange?: (checked: boolean) => void;
  labelPosition?: "left" | "right";
};

export default function Switch(props: SwitchProps) {
  const handleChange = (value: React.ChangeEvent<HTMLInputElement>) => {
    if (props.onChange) {
      props.onChange(value.target.checked);
    }
  };
  return (
    <label className={styles.switch}>
      <span className={styles.innerSwitch}>
        <input checked={props.checked} onChange={handleChange} type="checkbox" />
        <span className={classNames(styles.slider, styles.round)}></span>
      </span>
      <div
        className={classNames({
          [styles.left]: props.labelPosition && props.labelPosition === "left",
        })}
      >
        {props.label}
      </div>
    </label>
  );
}
