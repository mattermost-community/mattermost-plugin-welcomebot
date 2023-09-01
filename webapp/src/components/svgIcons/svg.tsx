import React from 'react';

const EditSvg = () => (
    <svg
        className='svg'
        xmlns='http://www.w3.org/2000/svg'
        width='16'
        height='16'
        viewBox='0 0 24 24'
        fill='none'
        stroke='#333333'
        strokeWidth='1.65'
        strokeLinecap='round'
        strokeLinejoin='round'
    >
        <path d='M20 14.66V20a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h5.34'/>
        <polygon points='18 2 22 6 12 16 8 16 8 12 18 2'/>
    </svg>
);

const DeleteSvg = () => (
    <svg
        className='svg'
        xmlns='http://www.w3.org/2000/svg'
        width='16'
        height='16'
        viewBox='0 0 24 24'
        fill='none'
        stroke='#333333'
        strokeWidth='1.65'
        strokeLinecap='round'
        strokeLinejoin='round'
    >
        <polyline points='3 6 5 6 21 6'/>
        <path d='M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2'/>
        <line
            x1='10'
            y1='11'
            x2='10'
            y2='17'
        />
        <line
            x1='14'
            y1='11'
            x2='14'
            y2='17'
        />
    </svg>
);

const ViewSvg = () => (
    <svg
        xmlns='http://www.w3.org/2000/svg'
        width='16'
        height='16'
        viewBox='0 0 24 24'
        fill='none'
        stroke='#333333'
        strokeWidth='1.65'
        strokeLinecap='round'
        strokeLinejoin='round'
    >
        <path d='M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z'/>
        <circle
            cx='12'
            cy='12'
            r='3'
        />
    </svg>
);

export {EditSvg, DeleteSvg, ViewSvg};
