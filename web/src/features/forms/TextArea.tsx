import styles from "./textInput.module.scss";
import React from "react";

interface textInputProps {
  label?: string;
  value?: string;
  placeholder?: string;
  className?: string;
  onChange?: (value: string) => void;
}
function TextArea(props: textInputProps) {
  const [localValue, setLocalValue] = React.useState("" as string | undefined);
  React.useEffect(() => {
    if (props.value === undefined) {
      return;
    }
    setLocalValue(props.value);
  }, [props.value]);
  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement | HTMLInputElement>) => {
    setLocalValue(e.target.value);
    props.onChange && props.onChange(e.target.value);
  };
  const inputId = "input-" + Math.random().toString(36).substr(2, 9);

  return (
    <div style={{ marginBottom: "var(--form-margin-bottom)" }}>
      <div style={{ marginBottom: "var(--form-label-margin-bottom)" }}>
        <label htmlFor={inputId}>{props.label}</label>
      </div>
      <textarea
        id={inputId}
        name={inputId}
        placeholder={props.placeholder}
        className={styles.textInput}
        value={localValue}
        onChange={handleChange}
      />
    </div>
  );
}
export default TextArea;
