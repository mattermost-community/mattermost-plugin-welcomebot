import React, {useState} from 'react';

import {FormGroup, Table, ButtonGroup, Button, OverlayTrigger, Tooltip} from 'react-bootstrap';

import ActionModal from '../modals/actionModal';
import DeleteModal from '../modals/deleteModal';
import ConfigModal from '../modals/configModal';

import {DeleteSvg, EditSvg, ViewSvg} from '../svgIcons/svg';

import './styles.scss';

type Props = {
    onChange: (config: Configs[]) => void;
    value: Configs[];
}

const ExistingConfigTable = ({value, onChange}: Props) => {
    const [isviewVisible, setIsViewVisible] = useState(false);
    const [isdeleteVisible, setIsDeleteVisible] = useState(false);
    const [iseditVisible, setIsEditVisible] = useState(false);
    const [isaddVisible, setIsAddVisible] = useState(false);

    const [configIndex, setConfigIndex] = useState(0);

    const handleView = (index: number) => {
        setConfigIndex(index);
        setIsViewVisible(true);
    };

    const handleDelete = (index: number) => {
        setConfigIndex(index);
        setIsDeleteVisible(true);
    };

    const handleEdit = (index: number) => {
        setConfigIndex(index);
        setIsEditVisible(true);
    };

    const handleAdd = () => setIsAddVisible(true);

    return (
        <div className='config'>
            {
                isviewVisible &&
                    <ActionModal
                        visibility={isviewVisible}
                        setVisibility={setIsViewVisible}
                        config={value}
                        configIndex={configIndex}
                    />
            }
            {isdeleteVisible &&
                <DeleteModal
                    visibility={isdeleteVisible}
                    setVisibility={setIsDeleteVisible}
                    config={value}
                    configIndex={configIndex}
                    onChange={onChange}
                />
            }
            {iseditVisible &&
                <ConfigModal
                    visibility={iseditVisible}
                    setVisibility={setIsEditVisible}
                    configIndex={configIndex}
                    config={value}
                    onChange={onChange}
                    modalHeader='Edit Config'
                />
            }
            {isaddVisible &&
                <ConfigModal
                    visibility={isaddVisible}
                    setVisibility={setIsAddVisible}
                    configIndex={null}
                    config={value}
                    onChange={onChange}
                    modalHeader='Add Config'
                />
            }
            <FormGroup>
                <div className='name'>
                    {'Existing Configs'}
                </div>
                <div>
                    {value.length > 0 &&
                    <Table
                        striped={true}
                        className='existing-config-table'
                    >
                        <thead>
                            <tr>
                                <th className='teamName'>{'Team Name'}</th>
                                <th className='delay'>{'Delay (in sec)'}</th>
                                <th className='message'>{'Message'}</th>
                                <th className='include-guests'>{'Include Guests'}</th>
                                <th className='option'>{'Options'}</th>
                            </tr>
                        </thead>
                        <tbody className='table-body'>
                            {
                                value.map((val, i) =>
                                    (
                                        <tr key={i.toString()}>
                                            <td>
                                                <OverlayTrigger
                                                    placement='top'
                                                    overlay={<Tooltip>{val.teamName}</Tooltip>}
                                                >
                                                    <p>
                                                        {val.teamName}
                                                    </p>
                                                </OverlayTrigger>
                                            </td>
                                            <td>{val.delayInSeconds}</td>
                                            <td className='message'>
                                                <OverlayTrigger
                                                    placement='top'
                                                    overlay={<Tooltip>{val.message.join(',')}</Tooltip>}
                                                >
                                                    <p className='message-content'>
                                                        {val.message.join(',')}
                                                    </p>
                                                </OverlayTrigger>
                                            </td>
                                            <td>{val.includeGuests ? val.includeGuests : '-'}</td>
                                            <td className='option'>
                                                <div>
                                                    <ButtonGroup aria-label='Basic example'>
                                                        <OverlayTrigger
                                                            placement='top'
                                                            overlay={<Tooltip>{'View actions'}</Tooltip>}
                                                        >
                                                            <Button
                                                                className='svg-buttons'
                                                                onClick={() => handleView(i)}
                                                            >
                                                                <ViewSvg/>
                                                            </Button>
                                                        </OverlayTrigger>
                                                        <OverlayTrigger
                                                            placement='top'
                                                            overlay={<Tooltip>{'Edit config'}</Tooltip>}
                                                        >
                                                            <Button
                                                                className='svg-buttons'
                                                                onClick={() => handleEdit(i)}
                                                            >
                                                                <EditSvg/>
                                                            </Button>
                                                        </OverlayTrigger>
                                                        <OverlayTrigger
                                                            placement='top'
                                                            overlay={<Tooltip>{'Delete Config'}</Tooltip>}
                                                        >
                                                            <Button
                                                                className='svg-buttons'
                                                                onClick={() => handleDelete(i)}
                                                            >
                                                                <DeleteSvg/>
                                                            </Button>
                                                        </OverlayTrigger>
                                                    </ButtonGroup>
                                                </div>
                                            </td>
                                        </tr>
                                    ),
                                )
                            }
                        </tbody>
                    </Table>}
                    <Button
                        className='add-config-btn'
                        variant='primary'
                        onClick={handleAdd}
                    >
                        {'Add Config'}
                    </Button>
                </div>
            </FormGroup>
        </div>
    );
};

export default ExistingConfigTable;
