import Box from "./Box";
import styles from "./NodeSettings.module.scss";
import Select, { SelectOption } from "../forms/Select";
import React, { useState } from "react";
import {
  Save20Regular as SaveIcon,
  Play16Regular as ConnectedIcon,
  Pause16Regular as DisconnectedIcon,
  ChevronDown20Regular as CollapsedIcon,
  LineHorizontal120Regular as ExpandedIcon,
  MoreCircle20Regular as MoreIcon,
  Delete24Regular as DeleteIconHeader,
  Delete20Regular as DeleteIcon,
  Pause20Regular as PauseIcon,
  Play20Regular as PlayIcon,
} from "@fluentui/react-icons";
import Spinny from "features/spinny/Spinny";
import { toastCategory } from "../toast/Toasts";
import ToastContext from "../toast/context";
import File from "../forms/File";
import TextInput from "features/forms/TextInput";
import {
  useGetLocalNodeQuery,
  useUpdateLocalNodeMutation,
  useAddLocalNodeMutation,
  useUpdateLocalNodeSetDisabledMutation,
  useUpdateLocalNodeSetDeletedMutation,
} from "apiSlice";
import { localNode } from "apiTypes";
import classNames from "classnames";
import Collapse from "features/collapse/Collapse";
import Popover from "features/popover/Popover";
import Button, { buttonColor, buttonPosition } from "features/buttons/Button";
import Modal from "features/modal/Modal";

interface nodeProps {
  localNodeId: number;
  collapsed?: boolean;
  addMode?: boolean;
  onAddSuccess?: () => void;
  onAddFailure?: () => void;
}

const NodeSettings = React.forwardRef(function NodeSettings(
  { localNodeId, collapsed, addMode, onAddSuccess }: nodeProps,
  ref
) {
  const toastRef = React.useContext(ToastContext);
  const popoverRef = React.useRef();

  const { data: localNodeData } = useGetLocalNodeQuery(localNodeId, {
    skip: !localNodeId,
  });
  const [updateLocalNode] = useUpdateLocalNodeMutation();
  const [addLocalNode] = useAddLocalNodeMutation();
  const [setDisableLocalNode] = useUpdateLocalNodeSetDisabledMutation();
  const [deleteLocalNode] = useUpdateLocalNodeSetDeletedMutation();

  const [localState, setLocalState] = useState({} as localNode);
  const [collapsedState, setCollapsedState] = useState(collapsed ?? false);
  const [showModalState, setShowModalState] = useState(false);
  const [deleteConfirmationTextInputState, setDeleteConfirmationTextInputState] = useState("");
  const [deleteEnabled, setDeleteEnabled] = useState(false);
  const [saveEnabledState, setSaveEnabledState] = useState(true);
  const [enableEnableButtonState, setEnableEnableButtonState] = useState(true);

  React.useImperativeHandle(ref, () => ({
    clear() {
      clear();
    },
  }));

  const clear = () => {
    setLocalState({ grpcAddress: "", implementation: "" } as localNode);
  };

  React.useEffect(() => {
    if (collapsed != undefined) {
      setCollapsedState(collapsed);
    }
  }, [collapsed]);

  const handleConfirmationModalClose = () => {
    setShowModalState(false);
    setDeleteConfirmationTextInputState("");
    setDeleteEnabled(false);
    setLocalState({} as localNode);
  };

  const handleDeleteClick = () => {
    if (popoverRef.current) {
      (popoverRef.current as { close: () => void }).close();
    }
    setShowModalState(true);
  };

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    submitNodeSettings();
  };

  const submitNodeSettings = async () => {
    setSaveEnabledState(false);
    const form = new FormData();
    form.append("implementation", "LND");
    form.append("grpcAddress", localState.grpcAddress ?? "");
    if (localState.tlsFile) {
      form.append("tlsFile", localState.tlsFile, localState.tlsFileName);
    }
    if (localState.macaroonFile) {
      form.append("macaroonFile", localState.macaroonFile, localState.macaroonFileName);
    }
    // we are adding new node
    if (!localState.localNodeId) {
      addLocalNode(form)
        .unwrap()
        .then((_) => {
          setSaveEnabledState(true);
          toastRef?.current?.addToast("Local node added", toastCategory.success);
          if (onAddSuccess) {
            onAddSuccess();
          }
        })
        .catch((error) => {
          setSaveEnabledState(true);
          toastRef?.current?.addToast(error.data["errors"]["server"][0].split(":")[0], toastCategory.error);
        });

      return;
    }
    updateLocalNode({ form, localNodeId: localState.localNodeId })
      .unwrap()
      .then((_) => {
        setSaveEnabledState(true);
        toastRef?.current?.addToast("Local node info saved", toastCategory.success);
      })
      .catch((error) => {
        setSaveEnabledState(true);
        toastRef?.current?.addToast(error.data["errors"]["server"][0].split(":")[0], toastCategory.error);
      });
  };

  React.useEffect(() => {
    if (localNodeData) {
      setLocalState(localNodeData);
    }
  }, [localNodeData]);

  const handleTLSFileChange = (file: File | null) => {
    setLocalState({ ...localState, tlsFile: file, tlsFileName: file ? file.name : undefined });
  };

  const handleMacaroonFileChange = (file: File | null) => {
    setLocalState({ ...localState, macaroonFile: file, macaroonFileName: file ? file.name : undefined });
  };

  const handleAddressChange = (value: string) => {
    setLocalState({ ...localState, grpcAddress: value });
  };

  const handleCollapseClick = () => {
    setCollapsedState(!collapsedState);
  };

  const handleModalDeleteClick = () => {
    setShowModalState(false);
    setDeleteConfirmationTextInputState("");
    setDeleteEnabled(false);
    deleteLocalNode({ localNodeId: localState.localNodeId });
  };

  const handleDeleteConfirmationTextInputChange = (value: string) => {
    setDeleteConfirmationTextInputState(value as string);
    setDeleteEnabled(value.toLowerCase() === "delete");
  };

  const handleDisableClick = () => {
    setEnableEnableButtonState(false);
    setDisableLocalNode({ localNodeId: localState.localNodeId, disabled: !localState.disabled })
      .unwrap()
      .finally(() => {
        setEnableEnableButtonState(true);
      });
    if (popoverRef.current) {
      (popoverRef.current as { close: () => void }).close();
    }
  };

  const implementationOptions = [{ value: "LND", label: "LND" } as SelectOption];

  const menuButton = <MoreIcon className={styles.moreIcon} />;
  return (
    <Box>
      <>
        {!addMode && (
          <div className={styles.header} onClick={handleCollapseClick}>
            <div
              className={classNames(styles.connectionIcon, {
                [styles.connected]: true,
                [styles.disabled]: localState.disabled,
              })}
            >
              {!localState.disabled && <ConnectedIcon />}
              {localState.disabled && <DisconnectedIcon />}
            </div>
            <div className={styles.title}>{localNodeData?.grpcAddress}</div>
            <div className={classNames(styles.collapseIcon, { [styles.collapsed]: collapsedState })}>
              {collapsedState ? <CollapsedIcon /> : <ExpandedIcon />}
            </div>
          </div>
        )}
        <Collapse collapsed={collapsedState} animate={!addMode}>
          <>
            {!addMode && (
              <>
                <div className={styles.borderSection}>
                  <div className={styles.detailHeader}>
                    <h4 className={styles.detailsTitle}>Node Details</h4>
                    <Popover button={menuButton} className={classNames("right", styles.moreButton)} ref={popoverRef}>
                      <div className={styles.nodeMenu}>
                        <Button
                          buttonColor={buttonColor.secondary}
                          text={localState.disabled ? "Enable node" : "Disable node"}
                          icon={localState.disabled ? <PlayIcon /> : <PauseIcon />}
                          onClick={handleDisableClick}
                          disabled={!enableEnableButtonState}
                        />
                        <Button
                          buttonColor={buttonColor.warning}
                          text={"Delete node"}
                          icon={<DeleteIcon />}
                          onClick={handleDeleteClick}
                        />
                      </div>
                    </Popover>
                  </div>
                </div>
              </>
            )}
            <div className={""}>
              <form onSubmit={handleSubmit}>
                <Select
                  label="Implementation"
                  onChange={() => {
                    return;
                  }}
                  options={implementationOptions}
                  value={implementationOptions.find((io) => io.value === localState.implementation)}
                />
                <span id="address">
                  <TextInput
                    label="GRPC Address (IP or Tor)"
                    value={localState.grpcAddress}
                    onChange={(e) => handleAddressChange(e as string)}
                    placeholder="100.100.100.100:10009"
                  />
                </span>
                <span id="tls">
                  <File label="TLS Certificate" onFileChange={handleTLSFileChange} fileName={localState?.tlsFileName} />
                </span>
                <span id="macaroon">
                  <File
                    label="Macaroon"
                    onFileChange={handleMacaroonFileChange}
                    fileName={localState?.macaroonFileName}
                  />
                </span>
                <Button
                  buttonColor={buttonColor.green}
                  text={addMode ? "Add Node" : saveEnabledState ? "Save node details" : "Saving..."}
                  icon={saveEnabledState ? <SaveIcon /> : <Spinny />}
                  onClick={submitNodeSettings}
                  buttonPosition={buttonPosition.fullWidth}
                  disabled={!saveEnabledState}
                />
              </form>
            </div>
          </>
        </Collapse>
        <Modal
          title={"Are you sure?"}
          icon={<DeleteIconHeader />}
          onClose={handleConfirmationModalClose}
          show={showModalState}
        >
          <div className={styles.deleteConfirm}>
            <p>
              Deleting the node will prevent you from viewing it&apos;s data in Torq. Alternatively set node to disabled
              to simply stop the data subscription but keep data collected so far.
            </p>
            <p>
              This operation cannot be undone, type &quot;<span className={styles.red}>delete</span>&quot; to confirm.
            </p>

            <TextInput
              value={deleteConfirmationTextInputState}
              onChange={(e) => handleDeleteConfirmationTextInputChange(e as string)}
            />
            <div className={styles.deleteConfirmButtons}>
              <Button
                buttonColor={buttonColor.warning}
                buttonPosition={buttonPosition.fullWidth}
                text={"Delete node"}
                icon={<DeleteIcon />}
                onClick={handleModalDeleteClick}
                disabled={!deleteEnabled}
              />
            </div>
          </div>
        </Modal>
      </>
    </Box>
  );
});
export default NodeSettings;
