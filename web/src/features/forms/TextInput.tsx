import styles from "./textInput.module.scss";
import React from "react";

interface textInputProps {
  label?: string;
  value?: string | number;
  placeholder?: string;
  inputType?: "text" | "number" | "email" | "password" | "search";
  onChange?: (value: string | number) => void;
}
function TextInput({ label, value, placeholder, inputType, onChange }: textInputProps) {
  const [localValue, setLocalValue] = React.useState<string | number | undefined>("");
  React.useEffect(() => {
    if (value === undefined) {
      return;
    }
    setLocalValue(value);
  }, [value]);
  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement | HTMLInputElement>) => {
    setLocalValue(e.target.value);
    onChange && onChange(e.target.value);
  };
  return (
    <div style={{ marginBottom: "var(--form-margin-bottom)" }}>
      <div style={{ marginBottom: "var(--form-label-margin-bottom)" }}>
        <span>{label}</span>
      </div>
      <input
        type={inputType}
        placeholder={placeholder}
        className={styles.textInput}
        value={localValue}
        onChange={handleChange}
      />
    </div>
  );
}
export default TextInput;
