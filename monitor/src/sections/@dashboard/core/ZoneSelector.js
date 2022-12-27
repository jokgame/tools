import * as React from 'react';
import PropTypes from 'prop-types';

import { FormControl, Autocomplete, TextField } from '@mui/material';

ZoneSelector.propTypes = {
  type: PropTypes.string,
  items: PropTypes.arrayOf(PropTypes.string),
  defaultValue: PropTypes.string,
  handleChange: PropTypes.func,
};

export default function ZoneSelector({ type, items, defaultValue, handleChange }) {
  const id = `${type}-select-zone`;

  const [zone, setZone] = React.useState(defaultValue || '');

  const onChange = (event) => {
    setZone(event.target.value);
    handleChange(event.target.value);
  };

  return (
    <FormControl fullWidth>
      <Autocomplete
        value={zone}
        id={id}
        options={items}
        onChange={onChange}
        renderInput={(params) => <TextField {...params} label="Zone" />}
      />
    </FormControl>
  );
}
