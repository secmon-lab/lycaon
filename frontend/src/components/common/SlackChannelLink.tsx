import React from 'react';
import { MessageSquare, ExternalLink } from 'lucide-react';

interface SlackChannelLinkProps {
  channelId: string;
  channelName: string;
  teamId?: string;
  showIcon?: boolean;
  className?: string;
}

const SlackChannelLink: React.FC<SlackChannelLinkProps> = ({
  channelId,
  channelName,
  teamId,
  showIcon = true,
  className = '',
}) => {
  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();

    const slackDeepLink = `slack://channel?team=${teamId}&id=${channelId}`;
    const webUrl = `https://app.slack.com/client/${teamId}/${channelId}`;

    let timeoutId: number;

    const visibilityChangeHandler = () => {
      if (document.visibilityState === 'hidden') {
        clearTimeout(timeoutId);
        window.removeEventListener('visibilitychange', visibilityChangeHandler);
      }
    };
    window.addEventListener('visibilitychange', visibilityChangeHandler);

    // Try to open deep link by creating and clicking a temporary link
    const link = document.createElement('a');
    link.href = slackDeepLink;
    link.style.display = 'none';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);

    timeoutId = window.setTimeout(() => {
      window.removeEventListener('visibilitychange', visibilityChangeHandler);
      // If the page is still visible after a delay, the deep link likely failed
      if (document.visibilityState === 'visible') {
        window.open(webUrl, '_blank', 'noopener,noreferrer');
      }
    }, 1000);
  };

  // If no teamId, render as plain text
  if (!teamId) {
    return (
      <span className={`inline-flex items-center gap-1 text-slate-600 ${className}`}>
        {showIcon && <MessageSquare className="h-3 w-3" />}
        #{channelName}
      </span>
    );
  }

  // Render as clickable link
  return (
    <a
      href={`https://app.slack.com/client/${teamId}/${channelId}`}
      onClick={handleClick}
      target="_blank"
      rel="noopener noreferrer"
      className={`inline-flex items-center gap-1 text-blue-600 hover:text-blue-800 hover:underline transition-colors ${className}`}
      title={`Open #${channelName} in Slack`}
    >
      {showIcon && <MessageSquare className="h-3 w-3" />}
      #{channelName}
      <ExternalLink className="h-3 w-3 opacity-60" />
    </a>
  );
};

export default SlackChannelLink;