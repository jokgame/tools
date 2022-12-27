import * as React from 'react';
import PropTypes from 'prop-types';

import { Select, FormControl, InputLabel, MenuItem } from '@mui/material';

TypeSelector.propTypes = {
  type: PropTypes.string,
  items: PropTypes.arrayOf(PropTypes.string),
  defaultValue: PropTypes.string,
  handleChange: PropTypes.func,
};

export default function TypeSelector({ type, items, defaultValue, handleChange }) {
  const labelId = `${type}-select-type-label`;
  const id = `${type}-select-type`;

  const [_type, setType] = React.useState(defaultValue || '');

  const onChange = (event) => {
    setType(event.target.value);
    handleChange(event.target.value);
  };

  return (
    <FormControl fullWidth>
      <InputLabel id={labelId}>Type</InputLabel>
      <Select
        labelId={labelId}
        id={id}
        value={_type}
        label="type"
        onChange={onChange}
        defaultValue={{ label: defaultValue, value: defaultValue }}
      >
        {items.map((item) => (
          <MenuItem key={item} value={item}>
            {item}
          </MenuItem>
        ))}
      </Select>
    </FormControl>
  );
}
