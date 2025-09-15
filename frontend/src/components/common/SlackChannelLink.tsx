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
    if (!teamId) {
      // If no team ID, just prevent default and do nothing
      e.preventDefault();
      return;
    }

    e.preventDefault();
    
    // Try to open in Slack desktop app first
    const slackDeepLink = `slack://channel?team=${teamId}&id=${channelId}`;
    
    // Create a temporary link element to open the deep link
    const link = document.createElement('a');
    link.href = slackDeepLink;
    link.style.display = 'none';
    document.body.appendChild(link);
    
    // Try to open the deep link
    link.click();
    
    // Clean up
    document.body.removeChild(link);
    
    // Set a timeout to open web version if desktop app doesn't respond
    setTimeout(() => {
      // If we're still on the same page, open web version
      if (document.hasFocus()) {
        const webUrl = `https://app.slack.com/client/${teamId}/${channelId}`;
        window.open(webUrl, '_blank');
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
      href="#"
      onClick={handleClick}
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