import styled from "@emotion/styled";
import Select, { Props, components } from "react-select";
import { ChevronDown16Regular as ChevronDownIcon } from "@fluentui/react-icons";

export type SelectOptionType = { value: string; label: string };

export const StyledSelect = styled(Select)`
  background: #ffffff;
`;

const customStyles = {
  indicatorSeparator: () => {
    return {};
  },
  control: (provided: any, _: any) => ({
    ...provided,
    borderRadius: 2,
    border: "1px solid var(--content-subtle)",
    boxShadow: "none",
    "&:hover": {
      border: "1px solid #8198a3",
      boxShadow: "none",
    },
  }),
  dropdownIndicator: (provided: any, _: any) => ({
    ...provided,
    color: "var(--content-default)",
    // padding: "0",
  }),
  singleValue: (provided: any) => ({
    ...provided,
    // fontSize: "var(--font-size-small)",
  }),
  option: (provided: any, state: any) => ({
    ...provided,
    color: "#8198a3",
    background: state.isFocused ? "var(--primary-150)" : "#ffffff",
    "&:hover": {
      boxShadow: "none",
      backgroundColor: "var(--primary-100)",
    },
    // fontSize: "var(--font-size-small)",
  }),
  menuList: (provided: any, _: any) => ({
    ...provided,
    background: "#f9fafb",
  }),
  menu: (provided: any, _: any) => ({
    ...provided,
    zIndex: "10",
  }),
};

export default function TorqSelect(props: Props) {
  const DropdownIndicator = (props: any) => {
    return (
      <components.DropdownIndicator {...props}>
        <ChevronDownIcon />
      </components.DropdownIndicator>
    );
  };
  return <StyledSelect components={{ DropdownIndicator }} styles={customStyles} {...props} />;
}
