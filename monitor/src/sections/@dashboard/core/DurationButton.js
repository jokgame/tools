import PropTypes from 'prop-types';
// @mui
import { Button } from '@mui/material';

// ----------------------------------------------------------------------

DurationButton.propTypes = {
  duration: PropTypes.object,
  onClick: PropTypes.func,
  selected: PropTypes.bool,
};

export default function DurationButton({ duration, onClick, selected }) {
  const minute = 60;
  const hour = 60 * minute;
  const day = 24 * hour;

  const getDurationDesc = () => {
    const total = duration.unit * duration.count;
    switch (total) {
      case 60:
        return '1 Minute';
      case 3600:
        return '1 Hour';
      case 3600 * 24:
        return '1 Day';
      default:
        if (total % day === 0) {
          return `${total / day} Days`;
        }
        if (total % hour === 0) {
          return `${total / hour} Hours`;
        }
        if (total % minute === 0) {
          return `${total / minute} Minutes`;
        }
        return `${total} Seconds`;
    }
  };

  return (
    <Button
      variant={selected ? 'contained' : 'outlined'}
      onClick={() => {
        onClick(duration);
      }}
    >
      {getDurationDesc()}
    </Button>
  );
}
